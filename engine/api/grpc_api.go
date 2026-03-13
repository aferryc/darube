package api

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"engine/store"

	"github.com/google/uuid"
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

type GRPCInvokeRequest struct {
	Service   string     `json:"service"` // fully qualified: package.Service
	Method    string     `json:"method"`  // method name (no slash)
	Request   string     `json:"request"` // JSON
	Headers   []KeyValue `json:"headers,omitempty"`
	Auth      *store.GRPCAuth `json:"auth,omitempty"` // if nil, use connection auth
	TimeoutMs int        `json:"timeout_ms,omitempty"`
}

type GRPCInvokeResponse struct {
	Success     bool                `json:"success"`
	Response    string              `json:"response,omitempty"` // JSON
	Headers     map[string][]string `json:"headers,omitempty"`
	Trailers    map[string][]string `json:"trailers,omitempty"`
	Error       string              `json:"error,omitempty"`
	DurationMs  float64             `json:"duration_ms,omitempty"`
}

type GRPCReflectResponse struct {
	Success  bool     `json:"success"`
	Services []string `json:"services,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// TestGRPCHandler handles POST /api/grpc/test
func TestGRPCHandler(w http.ResponseWriter, r *http.Request) {
	var cfg store.GRPCConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(cfg.Address) == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "address is required (host:port)"}, http.StatusOK)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	conn, err := dialGRPC(ctx, cfg)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	defer conn.Close()
	services, err := listGRPCServices(ctx, conn)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	msg := "gRPC reachable"
	if len(services) > 0 {
		msg = fmt.Sprintf("gRPC reachable (%d services via reflection)", len(services))
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: msg}, http.StatusOK)
}

// SaveGRPCHandler handles POST /api/grpc and PUT /api/grpc/{id}
func SaveGRPCHandler(w http.ResponseWriter, r *http.Request) {
	var cfg store.GRPCConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	pathID := r.PathValue("id")
	if pathID != "" {
		cfg.ID = pathID
	}
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}

	// Preserve folder_id on updates unless explicitly changed via the folder PATCH endpoint.
	if pathID != "" && cfg.FolderID == "" {
		if existing, err := store.GetGRPCConfig(pathID); err == nil && existing != nil {
			cfg.FolderID = existing.FolderID
		}
	}

	if strings.TrimSpace(cfg.ConnectionName) == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "connection_name is required"}, http.StatusOK)
		return
	}
	if strings.TrimSpace(cfg.Address) == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "address is required (host:port)"}, http.StatusOK)
		return
	}

	if err := store.WriteGRPCConnection(cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Failed to save gRPC config: " + err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: "gRPC connection saved", ID: cfg.ID}, http.StatusOK)
}

// GetGRPCHandler handles GET /api/grpc/{id}
func GetGRPCHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetGRPCConfig(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "config": cfg}, http.StatusOK)
}

// DeleteGRPCHandler handles DELETE /api/grpc/{id}
func DeleteGRPCHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	if err := store.DeleteGRPCConnection(id); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: "gRPC connection deleted"}, http.StatusOK)
}

// PatchGRPCFolderHandler handles PATCH /api/grpc/{id}/folder
func PatchGRPCFolderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	var req struct {
		FolderID string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetGRPCConfig(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}
	cfg.FolderID = req.FolderID
	if err := store.WriteGRPCConnection(*cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: "gRPC folder updated"}, http.StatusOK)
}

// GRPCReflectHandler handles POST /api/grpc/{id}/reflect
func GRPCReflectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, GRPCReflectResponse{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetGRPCConfig(id)
	if err != nil {
		sendJSONResponse(w, GRPCReflectResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	conn, err := dialGRPC(ctx, *cfg)
	if err != nil {
		sendJSONResponse(w, GRPCReflectResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	defer conn.Close()

	services, err := listGRPCServices(ctx, conn)
	if err != nil {
		sendJSONResponse(w, GRPCReflectResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	sendJSONResponse(w, GRPCReflectResponse{Success: true, Services: services}, http.StatusOK)
}

// GRPCInvokeHandler handles POST /api/grpc/{id}/invoke
func GRPCInvokeHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetGRPCConfig(id)
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	var req GRPCInvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "Invalid request"}, http.StatusBadRequest)
		return
	}
	service := strings.TrimSpace(req.Service)
	method := strings.TrimSpace(req.Method)
	if service == "" || method == "" {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "service and method are required"}, http.StatusOK)
		return
	}

	timeout := 30 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	conn, err := dialGRPC(ctx, *cfg)
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	defer conn.Close()

	// Resolve descriptors via reflection.
	files, err := resolveFilesForSymbol(ctx, conn, service+"."+method)
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}

	desc, err := files.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "resolved descriptor is not a service"}, http.StatusOK)
		return
	}
	m := svc.Methods().ByName(protoreflect.Name(method))
	if m == nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "method not found on service"}, http.StatusOK)
		return
	}
	if m.IsStreamingClient() || m.IsStreamingServer() {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "streaming methods are not supported yet"}, http.StatusOK)
		return
	}

	inMsg := dynamicpb.NewMessage(m.Input())
	rawJSON := strings.TrimSpace(req.Request)
	if rawJSON == "" {
		rawJSON = "{}"
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(rawJSON), inMsg); err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: "invalid JSON request: " + err.Error()}, http.StatusOK)
		return
	}

	outMsg := dynamicpb.NewMessage(m.Output())

	md := metadata.MD{}
	for _, kv := range req.Headers {
		if !kv.Enabled {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv.Key))
		if k == "" {
			continue
		}
		md.Append(k, kv.Value)
	}

	auth := cfg.Auth
	if req.Auth != nil {
		auth = *req.Auth
	}
	applyGRPCAuth(md, auth)

	ctx = metadata.NewOutgoingContext(ctx, md)

	var hdr, trl metadata.MD
	path := "/" + service + "/" + method
	t0 := time.Now()
	err = conn.Invoke(ctx, path, inMsg, outMsg, grpc.Header(&hdr), grpc.Trailer(&trl))
	dur := float64(time.Since(t0).Milliseconds())
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error(), DurationMs: dur}, http.StatusOK)
		return
	}

	outBytes, err := (protojson.MarshalOptions{Multiline: true, Indent: "  "}).Marshal(outMsg)
	if err != nil {
		sendJSONResponse(w, GRPCInvokeResponse{Success: false, Error: err.Error(), DurationMs: dur}, http.StatusOK)
		return
	}

	sendJSONResponse(w, GRPCInvokeResponse{
		Success:    true,
		Response:   string(outBytes),
		Headers:    mdToMap(hdr),
		Trailers:   mdToMap(trl),
		DurationMs: dur,
	}, http.StatusOK)
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

func listGRPCServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
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
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func applyGRPCAuth(md metadata.MD, auth store.GRPCAuth) {
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "none":
		return
	case "bearer":
		if strings.TrimSpace(auth.BearerToken) == "" {
			return
		}
		md.Set("authorization", "Bearer "+strings.TrimSpace(auth.BearerToken))
	case "basic":
		if auth.Username == "" {
			return
		}
		raw := auth.Username + ":" + auth.Password
		md.Set("authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
	default:
		return
	}
}

func mdToMap(md metadata.MD) map[string][]string {
	out := map[string][]string{}
	for k, v := range md {
		out[k] = append([]string{}, v...)
	}
	return out
}
