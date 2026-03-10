package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
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
