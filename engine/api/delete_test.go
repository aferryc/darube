package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestDeleteConnectionHandler(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()

	cfg := store.ConnectionConfig{
		ID:             "conn-1",
		ConnectionName: "Test",
		DBType:         "postgres",
		Host:           "localhost",
		Port:           5432,
		User:           "u",
		Password:       "p",
	}
	store.WriteConnection(cfg)

	req := httptest.NewRequest(http.MethodDelete, "/api/connections/conn-1", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	DeleteConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp CommandOutput
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success: %+v", resp)
	}

	_, err := store.GetConnection("conn-1")
	if err == nil {
		t.Error("connection should be deleted")
	}
}

func TestDeleteConnectionHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/connections/", nil)
	rr := httptest.NewRecorder()
	DeleteConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}
