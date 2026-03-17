package script

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"engine/store"

	"github.com/dop251/goja"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type httpRequestOpts struct {
	Method      string                 `json:"method"`
	URL         string                 `json:"url"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]string      `json:"query,omitempty"`
	Body        interface{}            `json:"body,omitempty"` // string or object
	TimeoutMs   int                    `json:"timeout_ms,omitempty"`
	Auth        *store.HTTPAuth        `json:"auth,omitempty"`
}

func installNetAPIs(vm *goja.Runtime, ctx context.Context) error {
	if err := installHTTP(vm, ctx); err != nil {
		return err
	}
	if err := installGRPC(vm, ctx); err != nil {
		return err
	}
	return nil
}

func installHTTP(vm *goja.Runtime, ctx context.Context) error {
	httpObj := vm.NewObject()

	requestFn := func(baseURL string, baseAuth store.HTTPAuth, call goja.FunctionCall) goja.Value {
		var opts httpRequestOpts
		if len(call.Arguments) == 0 {
			panic(vm.NewGoError(fmt.Errorf("http.request(opts): opts is required")))
		}
		b, _ := json.Marshal(call.Argument(0).Export())
		if err := json.Unmarshal(b, &opts); err != nil {
			panic(vm.NewGoError(err))
		}

		method := strings.ToUpper(strings.TrimSpace(opts.Method))
		if method == "" {
			method = http.MethodGet
		}
		rawURL := strings.TrimSpace(opts.URL)
		if rawURL == "" {
			rawURL = baseURL
		}
		target, err := resolveURL(baseURL, rawURL)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		u, _ := url.Parse(target)
		q := u.Query()
		for k, v := range opts.QueryParams {
			if strings.TrimSpace(k) == "" {
				continue
			}
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()

		headers := map[string]string{}
		for k, v := range opts.Headers {
			headers[k] = v
		}

		auth := baseAuth
		if opts.Auth != nil {
			auth = *opts.Auth
		}
		applyHTTPAuth(headers, auth)

		var bodyReader io.Reader
		if opts.Body != nil {
			switch v := opts.Body.(type) {
			case string:
				bodyReader = strings.NewReader(v)
			default:
				enc, err := json.Marshal(v)
				if err != nil {
					panic(vm.NewGoError(err))
				}
				bodyReader = strings.NewReader(string(enc))
				ensureHeader(headers, "Content-Type", "application/json")
			}
		}

		timeout := 30 * time.Second
		if opts.TimeoutMs > 0 {
			timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
		}

		req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Do(req)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))

		out := map[string]interface{}{
			"status":      resp.StatusCode,
			"status_text": resp.Status,
			"headers":     resp.Header,
			"body_text":   string(body),
		}
		return vm.ToValue(out)
	}

	_ = httpObj.Set("request", func(call goja.FunctionCall) goja.Value {
		return requestFn("", store.HTTPAuth{Type: "none"}, call)
	})

	_ = httpObj.Set("conn", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		if id == "" || id == "undefined" || id == "null" {
			panic(vm.NewGoError(fmt.Errorf("http.conn(id): id is required")))
		}
		cfg, err := getHTTPConfigByIDOrName(id)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		obj := vm.NewObject()
		_ = obj.Set("id", cfg.ID)
		_ = obj.Set("baseURL", cfg.BaseURL)
		_ = obj.Set("request", func(call goja.FunctionCall) goja.Value {
			return requestFn(cfg.BaseURL, cfg.Auth, call)
		})
		return obj
	})

	return vm.Set("http", httpObj)
}

func installGRPC(vm *goja.Runtime, ctx context.Context) error {
	grpcObj := vm.NewObject()

	_ = grpcObj.Set("reflect", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewGoError(fmt.Errorf("grpc.reflect(opts): opts is required")))
		}
		var cfg store.GRPCConfig
		b, _ := json.Marshal(call.Argument(0).Export())
		if err := json.Unmarshal(b, &cfg); err != nil {
			panic(vm.NewGoError(err))
		}
		conn, err := dialGRPC(ctx, cfg)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer conn.Close()
		svcs, err := listServices(ctx, conn)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(svcs)
	})

	_ = grpcObj.Set("conn", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		if id == "" || id == "undefined" || id == "null" {
			panic(vm.NewGoError(fmt.Errorf("grpc.conn(id): id is required")))
		}
		cfg, err := getGRPCConfigByIDOrName(id)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		obj := vm.NewObject()
		_ = obj.Set("id", cfg.ID)
		_ = obj.Set("address", cfg.Address)
		_ = obj.Set("invoke", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) == 0 {
				panic(vm.NewGoError(fmt.Errorf("grpc.conn(id).invoke(opts): opts is required")))
			}
			opts := call.Argument(0).Export()
			b, _ := json.Marshal(opts)
			var req struct {
				Service string            `json:"service"`
				Method  string            `json:"method"`
				Request json.RawMessage   `json:"request"`
				Headers map[string]string `json:"headers,omitempty"`
				Auth    *store.GRPCAuth   `json:"auth,omitempty"`
			}
			if err := json.Unmarshal(b, &req); err != nil {
				panic(vm.NewGoError(err))
			}
			resp, err := grpcInvoke(ctx, *cfg, req.Service, req.Method, req.Request, req.Headers, req.Auth)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(resp)
		})
		return obj
	})

	return vm.Set("grpc", grpcObj)
}

func getHTTPConfigByIDOrName(idOrName string) (*store.HTTPConfig, error) {
	if cfg, err := store.GetHTTPConfig(idOrName); err == nil && cfg != nil {
		return cfg, nil
	}
	conns, err := store.ReadHTTPConnections()
	if err != nil {
		return nil, err
	}
	var match *store.HTTPConfig
	for i := range conns {
		if conns[i].ConnectionName == idOrName {
			if match != nil {
				return nil, fmt.Errorf("multiple HTTP connections share the name '%s' (use id instead)", idOrName)
			}
			match = &conns[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("http connection '%s' not found", idOrName)
	}
	return match, nil
}

func getGRPCConfigByIDOrName(idOrName string) (*store.GRPCConfig, error) {
	if cfg, err := store.GetGRPCConfig(idOrName); err == nil && cfg != nil {
		return cfg, nil
	}
	conns, err := store.ReadGRPCConnections()
	if err != nil {
		return nil, err
	}
	var match *store.GRPCConfig
	for i := range conns {
		if conns[i].ConnectionName == idOrName {
			if match != nil {
				return nil, fmt.Errorf("multiple gRPC connections share the name '%s' (use id instead)", idOrName)
			}
			match = &conns[i]
		}
	}
	if match == nil {
		return nil, fmt.Errorf("grpc connection '%s' not found", idOrName)
	}
	return match, nil
}

func resolveURL(baseURL, raw string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		base, err := url.Parse(baseURL)
		if err == nil {
			u, err := url.Parse(raw)
			if err != nil {
				return "", err
			}
			if u.IsAbs() {
				return u.String(), nil
			}
			return base.ResolveReference(u).String(), nil
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func applyHTTPAuth(headers map[string]string, auth store.HTTPAuth) {
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "none":
		return
	case "bearer":
		if strings.TrimSpace(auth.BearerToken) == "" {
			return
		}
		headers["Authorization"] = "Bearer " + strings.TrimSpace(auth.BearerToken)
	case "basic":
		if auth.Username == "" {
			return
		}
		raw := auth.Username + ":" + auth.Password
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
	default:
		return
	}
}

func ensureHeader(headers map[string]string, key, value string) {
	for k := range headers {
		if strings.EqualFold(k, key) {
			return
		}
	}
	headers[key] = value
}

func dialGRPC(ctx context.Context, cfg store.GRPCConfig) (*grpc.ClientConn, error) {
	addr := strings.TrimSpace(cfg.Address)
	if addr == "" {
		return nil, fmt.Errorf("address is required")
	}
	var creds credentials.TransportCredentials
	if cfg.TLS {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.InsecureTLS,
		}
		if strings.TrimSpace(cfg.ServerName) != "" {
			tlsCfg.ServerName = strings.TrimSpace(cfg.ServerName)
		}
		creds = credentials.NewTLS(tlsCfg)
	} else {
		creds = insecure.NewCredentials()
	}
	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(creds), grpc.WithBlock())
}

func listServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
	c := grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	stream, err := c.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}); err != nil {
		return nil, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	ls := resp.GetListServicesResponse()
	if ls == nil {
		if e := resp.GetErrorResponse(); e != nil {
			return nil, fmt.Errorf("reflection error: %s", e.ErrorMessage)
		}
		return nil, fmt.Errorf("reflection: unexpected response")
	}
	out := make([]string, 0, len(ls.Service))
	for _, s := range ls.Service {
		if s.GetName() != "" {
			out = append(out, s.GetName())
		}
	}
	return out, nil
}

func resolveFilesForSymbol(ctx context.Context, conn *grpc.ClientConn, symbol string) (*protoregistry.Files, error) {
	c := grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	stream, err := c.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}); err != nil {
		return nil, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		if e := resp.GetErrorResponse(); e != nil {
			return nil, fmt.Errorf("reflection error: %s", e.ErrorMessage)
		}
		return nil, fmt.Errorf("reflection: unexpected response")
	}

	set := &descriptorpb.FileDescriptorSet{}
	for _, b := range fdResp.FileDescriptorProto {
		var fd descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(b, &fd); err != nil {
			return nil, err
		}
		set.File = append(set.File, &fd)
	}
	return protodesc.NewFiles(set)
}

func grpcInvoke(ctx context.Context, cfg store.GRPCConfig, service, method string, reqJSON json.RawMessage, headers map[string]string, authOverride *store.GRPCAuth) (map[string]interface{}, error) {
	conn, err := dialGRPC(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	files, err := resolveFilesForSymbol(ctx, conn, service+"."+method)
	if err != nil {
		return nil, err
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return nil, err
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("resolved descriptor is not a service")
	}
	m := svc.Methods().ByName(protoreflect.Name(method))
	if m == nil {
		return nil, fmt.Errorf("method not found on service")
	}
	if m.IsStreamingClient() || m.IsStreamingServer() {
		return nil, fmt.Errorf("streaming methods are not supported yet")
	}

	inMsg := dynamicpb.NewMessage(m.Input())
	if len(reqJSON) == 0 {
		reqJSON = []byte("{}")
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(reqJSON, inMsg); err != nil {
		return nil, fmt.Errorf("invalid JSON request: %w", err)
	}
	outMsg := dynamicpb.NewMessage(m.Output())

	md := metadata.MD{}
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		md.Set(strings.ToLower(strings.TrimSpace(k)), v)
	}

	auth := cfg.Auth
	if authOverride != nil {
		auth = *authOverride
	}
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "none":
	case "bearer":
		if strings.TrimSpace(auth.BearerToken) != "" {
			md.Set("authorization", "Bearer "+strings.TrimSpace(auth.BearerToken))
		}
	case "basic":
		if auth.Username != "" {
			raw := auth.Username + ":" + auth.Password
			md.Set("authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
		}
	}

	ctx = metadata.NewOutgoingContext(ctx, md)
	var hdr, trl metadata.MD
	path := "/" + service + "/" + method
	if err := conn.Invoke(ctx, path, inMsg, outMsg, grpc.Header(&hdr), grpc.Trailer(&trl)); err != nil {
		return nil, err
	}
	outBytes, err := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(outMsg)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"response": string(outBytes),
		"headers":  hdr,
		"trailers": trl,
	}, nil
}

