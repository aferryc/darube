package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"engine/script"
	"engine/store"
)

type ScriptRunRequest struct {
	Script    string `json:"script"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

type ScriptRunResponse struct {
	Success    bool        `json:"success"`
	Result     interface{} `json:"result,omitempty"`
	Logs       []string    `json:"logs,omitempty"`
	Error      string      `json:"error,omitempty"`
	DurationMS float64     `json:"duration_ms,omitempty"`
}

// RunScriptHandler handles POST /api/scripts/run
func RunScriptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScriptRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, ScriptRunResponse{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	if req.Script == "" {
		sendJSONResponse(w, ScriptRunResponse{Success: false, Error: "script is required"}, http.StatusBadRequest)
		return
	}

	// Apply global script timeout policy.
	settings, _ := store.LoadSettings()
	baseTimeout := time.Duration(0)
	if req.TimeoutMS > 0 {
		baseTimeout = time.Duration(req.TimeoutMS) * time.Millisecond
	} else {
		baseTimeout = 15 * time.Second
	}
	// Clamp to global maximum if configured and positive.
	if settings.GlobalScriptTimeoutMs > 0 {
		max := time.Duration(settings.GlobalScriptTimeoutMs) * time.Millisecond
		if baseTimeout <= 0 || baseTimeout > max {
			baseTimeout = max
		}
	}

	var ctx context.Context
	var cancel context.CancelFunc
	// Negative global timeout means "no limit" – use request context directly.
	if settings.GlobalScriptTimeoutMs < 0 {
		ctx = r.Context()
		cancel = func() {}
	} else {
		if baseTimeout <= 0 {
			baseTimeout = 15 * time.Second
		}
		ctx, cancel = context.WithTimeout(r.Context(), baseTimeout)
	}
	defer cancel()

	engine := script.NewDefault()
	t0 := time.Now()
	result, logs, err := engine.RunWithOutput(ctx, req.Script)
	dur := float64(time.Since(t0).Microseconds()) / 1000.0

	if err != nil {
		sendJSONResponse(w, ScriptRunResponse{Success: false, Error: err.Error(), Logs: logs, DurationMS: dur}, http.StatusOK)
		return
	}

	// Ensure response is JSON-marshalable; otherwise, stringify.
	resp := ScriptRunResponse{Success: true, Result: result, Logs: logs, DurationMS: dur}
	if _, mErr := json.Marshal(resp); mErr != nil {
		resp.Result = fmt.Sprint(result)
	}
	sendJSONResponse(w, resp, http.StatusOK)
}
