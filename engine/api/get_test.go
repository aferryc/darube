package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestGetConnectionHandler(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()

	cfg := store.ConnectionConfig{
		ID:             "conn-1",
		ConnectionName: "Prod",
		DBType:         "postgres",
		Host:           "localhost",
		Port:           5432,
		User:           "u",
		Password:       "secret",
	}
	store.WriteConnection(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	GetConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp store.ConnectionConfig
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "conn-1" || resp.ConnectionName != "Prod" {
		t.Errorf("unexpected config: %+v", resp)
	}
	if resp.Password != "" {
		t.Error("password should be stripped from response")
	}
}

func TestGetConnectionHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/", nil)
	rr := httptest.NewRecorder()
	GetConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetConnectionHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	GetConnectionHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d", rr.Code)
	}
}
