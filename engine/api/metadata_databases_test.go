package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestGetMetadataDatabasesHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/metadata/databases", nil)
	rr := httptest.NewRecorder()
	GetMetadataDatabasesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataDatabasesHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/metadata/databases", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	GetMetadataDatabasesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

