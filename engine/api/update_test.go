package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestUpdateConnectionHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/connections/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	UpdateConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestUpdateConnectionHandler_InvalidBody(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPut, "/api/connections/conn-1", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	UpdateConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestUpdateConnectionHandler_InvalidConnection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()

	// Save a connection first
	store.WriteConnection(store.ConnectionConfig{
		ID:       "conn-1",
		DBType:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "u",
		Password: "p",
	})

	// Try to update with unreachable host - will fail at db.Connect (connection refused)
	body, _ := json.Marshal(map[string]interface{}{
		"connection_name": "Updated",
		"db_type":         "postgres",
		"host":            "127.0.0.1",
		"port":            19999, // Unlikely to have postgres
		"user":            "u",
		"password":        "p",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/connections/conn-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	UpdateConnectionHandler(rr, req)

	// Should return 200 with success: false (connection failed)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp CommandOutput
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Error("expected success false for invalid connection")
	}
}
