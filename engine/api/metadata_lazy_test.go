package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestGetMetadataSchemasHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/metadata/schemas", nil)
	rr := httptest.NewRecorder()
	GetMetadataSchemasHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataSchemasHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/metadata/schemas", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	GetMetadataSchemasHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataTablesHandler_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/c1/metadata/schemas/public/tables", nil)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	GetMetadataTablesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataColumnsHandler_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/c1/metadata/schemas/s/tables/t/columns", nil)
	req.SetPathValue("id", "")
	req.SetPathValue("schema", "s")
	req.SetPathValue("table", "t")
	rr := httptest.NewRecorder()
	GetMetadataColumnsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataColumnsHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/metadata/schemas/main/tables/test/columns", nil)
	req.SetPathValue("id", "conn-1")
	req.SetPathValue("schema", "main")
	req.SetPathValue("table", "test")
	rr := httptest.NewRecorder()
	GetMetadataColumnsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}
