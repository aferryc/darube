package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"

	_ "modernc.org/sqlite"
)

func TestRefreshConnectionHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/refresh", nil)
	rr := httptest.NewRecorder()
	RefreshConnectionHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestRefreshConnectionHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/connections/nonexistent/refresh", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	RefreshConnectionHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp CommandOutput
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Success {
		t.Error("expected success false")
	}
}

func TestRefreshConnectionHandler_Success_SQLite(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	connID := "conn-refresh"
	if err := store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "sqlite", ConnectionName: "m", FilePath: ":memory:"}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/refresh", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	RefreshConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp CommandOutput
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %#v", resp)
	}
	if !store.IsConnected(connID) {
		t.Fatalf("expected connection to be active after refresh")
	}
}

func TestRefreshConnectionHandler_ConnectError(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	connID := "conn-refresh-bad"
	if err := store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "duckdb", ConnectionName: "bad"}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/refresh", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	RefreshConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp CommandOutput
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Success || resp.Error == "" {
		t.Fatalf("expected error: %#v", resp)
	}
}
