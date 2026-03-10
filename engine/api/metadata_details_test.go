package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestGetMetadataTableDMLHandler_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/c1/metadata/schemas/s/tables/t/dml", nil)
	req.SetPathValue("id", "c1")
	req.SetPathValue("schema", "")
	req.SetPathValue("table", "t")
	rr := httptest.NewRecorder()
	GetMetadataTableDMLHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataTableDMLHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/metadata/schemas/main/tables/test/dml", nil)
	req.SetPathValue("id", "conn-1")
	req.SetPathValue("schema", "main")
	req.SetPathValue("table", "test")
	rr := httptest.NewRecorder()
	GetMetadataTableDMLHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataTableIndexesHandler_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/c1/metadata/schemas/s/tables/t/indexes", nil)
	req.SetPathValue("id", "c1")
	req.SetPathValue("schema", "s")
	req.SetPathValue("table", "")
	rr := httptest.NewRecorder()
	GetMetadataTableIndexesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetMetadataTableIndexesHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/metadata/schemas/main/tables/test/indexes", nil)
	req.SetPathValue("id", "conn-1")
	req.SetPathValue("schema", "main")
	req.SetPathValue("table", "test")
	rr := httptest.NewRecorder()
	GetMetadataTableIndexesHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}
