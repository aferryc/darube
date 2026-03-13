package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"engine/store"

	"github.com/google/uuid"
)

type KeyValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type HTTPRequestBody struct {
	Type string `json:"type"` // "none" | "json" | "raw"
	Text string `json:"text"`
}

type HTTPRequest struct {
	Method      string          `json:"method"`
	URL         string          `json:"url"`
	QueryParams []KeyValue      `json:"query_params,omitempty"`
	Headers     []KeyValue      `json:"headers,omitempty"`
	Body        HTTPRequestBody `json:"body,omitempty"`
	Auth        *store.HTTPAuth `json:"auth,omitempty"` // if nil, use connection auth
	TimeoutMs   int             `json:"timeout_ms,omitempty"`
}

type HTTPResponse struct {
	Success    bool                `json:"success"`
	Status     int                 `json:"status,omitempty"`
	StatusText string              `json:"status_text,omitempty"`
	Headers    map[string][]string `json:"headers,omitempty"`
	BodyText   string              `json:"body_text,omitempty"`
	Error      string              `json:"error,omitempty"`
	DurationMs float64             `json:"duration_ms,omitempty"`
}

// TestHTTPHandler handles POST /api/http/test
func TestHTTPHandler(w http.ResponseWriter, r *http.Request) {
	var cfg store.HTTPConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "base_url is required"}, http.StatusOK)
		return
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "base_url must be a valid URL (e.g. https://api.example.com)"}, http.StatusOK)
		return
	}

	ctx := r.Context()
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	_ = resp.Body.Close()
	sendJSONResponse(w, CommandOutput{Success: true, Message: "HTTP reachable"}, http.StatusOK)
}

// SaveHTTPHandler handles POST /api/http and PUT /api/http/{id}
func SaveHTTPHandler(w http.ResponseWriter, r *http.Request) {
	var cfg store.HTTPConfig
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
		if existing, err := store.GetHTTPConfig(pathID); err == nil && existing != nil {
			cfg.FolderID = existing.FolderID
		}
	}

	if strings.TrimSpace(cfg.ConnectionName) == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "connection_name is required"}, http.StatusOK)
		return
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "base_url is required"}, http.StatusOK)
		return
	}
	if u, err := url.Parse(strings.TrimSpace(cfg.BaseURL)); err != nil || u.Scheme == "" || u.Host == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "base_url must be a valid URL (e.g. https://api.example.com)"}, http.StatusOK)
		return
	}

	// Normalize headers.
	for i := range cfg.DefaultHeaders {
		cfg.DefaultHeaders[i].Key = strings.TrimSpace(cfg.DefaultHeaders[i].Key)
	}

	if err := store.WriteHTTPConnection(cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Failed to save HTTP config: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, CommandOutput{Success: true, Message: "HTTP connection saved", ID: cfg.ID}, http.StatusOK)
}

// GetHTTPHandler handles GET /api/http/{id}
func GetHTTPHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetHTTPConfig(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "config": cfg}, http.StatusOK)
}

// DeleteHTTPHandler handles DELETE /api/http/{id}
func DeleteHTTPHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	if err := store.DeleteHTTPConnection(id); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: "HTTP connection deleted"}, http.StatusOK)
}

// PatchHTTPFolderHandler handles PATCH /api/http/{id}/folder
func PatchHTTPFolderHandler(w http.ResponseWriter, r *http.Request) {
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
	cfg, err := store.GetHTTPConfig(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}
	cfg.FolderID = req.FolderID
	if err := store.WriteHTTPConnection(*cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, CommandOutput{Success: true, Message: "HTTP folder updated"}, http.StatusOK)
}

// HTTPRequestHandler handles POST /api/http/{id}/request
func HTTPRequestHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}
	cfg, err := store.GetHTTPConfig(id)
	if err != nil {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	var req HTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: "Invalid request"}, http.StatusBadRequest)
		return
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	targetURL, err := resolveHTTPURL(cfg.BaseURL, strings.TrimSpace(req.URL))
	if err != nil {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	u, _ := url.Parse(targetURL)
	q := u.Query()
	for _, kv := range req.QueryParams {
		if !kv.Enabled {
			continue
		}
		k := strings.TrimSpace(kv.Key)
		if k == "" {
			continue
		}
		q.Set(k, kv.Value)
	}
	u.RawQuery = q.Encode()

	headers := map[string]string{}
	for _, h := range cfg.DefaultHeaders {
		if !h.Enabled {
			continue
		}
		k := strings.TrimSpace(h.Key)
		if k == "" {
			continue
		}
		headers[k] = h.Value
	}
	for _, kv := range req.Headers {
		if !kv.Enabled {
			continue
		}
		k := strings.TrimSpace(kv.Key)
		if k == "" {
			continue
		}
		headers[k] = kv.Value
	}

	auth := cfg.Auth
	if req.Auth != nil {
		auth = *req.Auth
	}
	applyHTTPAuth(headers, auth)

	var bodyReader io.Reader
	if strings.ToLower(strings.TrimSpace(req.Body.Type)) == "json" {
		bodyReader = strings.NewReader(req.Body.Text)
		ensureHeader(headers, "Content-Type", "application/json")
	} else if strings.ToLower(strings.TrimSpace(req.Body.Type)) == "raw" {
		bodyReader = strings.NewReader(req.Body.Text)
	} else {
		bodyReader = nil
	}

	ctx := r.Context()
	httpReq, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	timeout := 30 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}

	t0 := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		sendJSONResponse(w, HTTPResponse{Success: false, Error: err.Error(), DurationMs: float64(time.Since(t0).Milliseconds())}, http.StatusOK)
		return
	}
	defer resp.Body.Close()

	const maxBody = 5 * 1024 * 1024
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	truncated := len(body) > maxBody
	if truncated {
		body = body[:maxBody]
	}
	bodyText := string(body)
	if truncated {
		bodyText += "\n\n[Darube] Response truncated."
	}

	sendJSONResponse(w, HTTPResponse{
		Success:    true,
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    resp.Header,
		BodyText:   bodyText,
		DurationMs: float64(time.Since(t0).Milliseconds()),
	}, http.StatusOK)
}

func resolveHTTPURL(baseURL, raw string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("connection base_url is empty")
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return base.String(), nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.IsAbs() {
		return u.String(), nil
	}
	return base.ResolveReference(u).String(), nil
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

// Defensive: some servers require a body for certain methods, ensure we don't accidentally reuse buffers.
func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// Used by future enhancements; keep to avoid deadcode churn with upcoming request editor.
func bufferString(s string) *bytes.Buffer { return bytes.NewBufferString(s) }
