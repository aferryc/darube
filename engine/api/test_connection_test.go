package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
