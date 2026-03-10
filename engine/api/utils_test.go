package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendJSONResponse_Success(t *testing.T) {
	out := CommandOutput{Success: true, Message: "OK", ID: "conn-1"}
	rr := httptest.NewRecorder()
	sendJSONResponse(rr, out, http.StatusOK)

	if rr.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}

	var parsed CommandOutput
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if !parsed.Success || parsed.Message != "OK" || parsed.ID != "conn-1" {
		t.Errorf("unexpected body: %+v", parsed)
	}
}

func TestSendJSONResponse_ErrorStatus(t *testing.T) {
	out := map[string]interface{}{"success": false, "error": "not found"}
	rr := httptest.NewRecorder()
	sendJSONResponse(rr, out, http.StatusNotFound)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status code: got %d, want %d", rr.Code, http.StatusNotFound)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if parsed["success"] != false || parsed["error"] != "not found" {
		t.Errorf("unexpected body: %+v", parsed)
	}
}

func TestSendJSONResponse_MapPayload(t *testing.T) {
	out := map[string]interface{}{
		"success": true,
		"connections": []interface{}{
			map[string]interface{}{"id": "c1", "connection_name": "Prod"},
		},
	}
	rr := httptest.NewRecorder()
	sendJSONResponse(rr, out, http.StatusOK)

	if rr.Code != http.StatusOK {
		t.Errorf("status code: got %d", rr.Code)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	conns, ok := parsed["connections"].([]interface{})
	if !ok || len(conns) != 1 {
		t.Errorf("unexpected connections: %+v", parsed)
	}
}
