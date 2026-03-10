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

func TestDisconnectHandler_NotConnected(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/disconnect", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	DisconnectHandler(rr, req)

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
}

func TestDisconnectHandler_Connected(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store.AddActiveConnection("conn-1", db)

	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/disconnect", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	DisconnectHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	if store.IsConnected("conn-1") {
		t.Error("connection should be disconnected")
	}
}

func TestDisconnectHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections//disconnect", nil)
	rr := httptest.NewRecorder()
	DisconnectHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}
