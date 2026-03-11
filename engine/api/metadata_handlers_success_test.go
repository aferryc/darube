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

func TestMetadataHandlers_SqliteSuccess(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "connections.json"))
	defer restore()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	connID := "meta-sqlite"
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER, name TEXT);`); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "sqlite", ConnectionName: "m", FilePath: ":memory:"}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	// Databases
	req := httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/databases", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	GetMetadataDatabasesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("databases: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	// Schemas
	req = httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/schemas", nil)
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	GetMetadataSchemasHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("schemas: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var schemasResp LazySchemasResponse
	_ = json.NewDecoder(rr.Body).Decode(&schemasResp)
	if !schemasResp.Success || len(schemasResp.Schemas) != 1 || schemasResp.Schemas[0].Name != "main" {
		t.Fatalf("schemas resp: %#v", schemasResp)
	}

	// Tables
	req = httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/schemas/main/tables", nil)
	req.SetPathValue("id", connID)
	req.SetPathValue("schema", "main")
	rr = httptest.NewRecorder()
	GetMetadataTablesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("tables: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var tablesResp LazyTablesResponse
	_ = json.NewDecoder(rr.Body).Decode(&tablesResp)
	if !tablesResp.Success || len(tablesResp.Tables) == 0 {
		t.Fatalf("tables resp: %#v", tablesResp)
	}

	// Columns
	req = httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/schemas/main/tables/users/columns", nil)
	req.SetPathValue("id", connID)
	req.SetPathValue("schema", "main")
	req.SetPathValue("table", "users")
	rr = httptest.NewRecorder()
	GetMetadataColumnsHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("columns: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var colsResp LazyColumnsResponse
	_ = json.NewDecoder(rr.Body).Decode(&colsResp)
	if !colsResp.Success || len(colsResp.Columns) < 2 {
		t.Fatalf("columns resp: %#v", colsResp)
	}
}

