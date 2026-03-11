package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTestConnectionHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/test", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	TestConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp CommandOutput
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Error("expected success false")
	}
	if resp.Error != "Invalid request body" {
		t.Errorf("error: %s", resp.Error)
	}
}

func TestTestConnectionHandler_MissingRequiredFields(t *testing.T) {
	body := bytes.NewBufferString(`{"connection_name":"","db_type":"","host":"","user":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	TestConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp CommandOutput
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Error("expected success false")
	}
	if resp.Error == "" {
		t.Error("expected error message")
	}
}

func TestTestConnectionHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/test", nil)
	rr := httptest.NewRecorder()
	TestConnectionHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestTestConnectionHandler_SQLiteSuccess(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"connection_name": "mem",
		"db_type":         "sqlite",
		"file_path":       ":memory:",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	TestConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp CommandOutput
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %#v", resp)
	}
}

func TestTestConnectionHandler_OracleValidation(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"connection_name": "ora",
		"db_type":         "oracle",
		"host":            "",
		"port":            0,
		"user":            "",
		"dbname":          "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	TestConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
