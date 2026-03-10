package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"

	_ "modernc.org/sqlite"
)

func setupExplainTest(t *testing.T) (string, func()) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE test (id INTEGER)")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	connID := "conn-explain-test"
	store.WriteConnection(store.ConnectionConfig{
		ID:     connID,
		DBType: "sqlite",
		Host:   "localhost",
		Port:   0,
		User:   "",
		Password: "",
	})
	store.AddActiveConnection(connID, db)
	return connID, func() {
		store.RemoveActiveConnection(connID)
		restore()
	}
}

func TestExplainHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExplainHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/explain", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExplainHandler_EmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/explain", bytes.NewBufferString(`{"query":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExplainHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExplainHandler_SqlServerNotSupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()
	// Add sqlserver config (no real connection needed for this path)
	store.WriteConnection(store.ConnectionConfig{
		ID:     "conn-sqlserver",
		DBType: "sqlserver",
		Host:   "localhost",
		Port:   1433,
		User:   "sa",
		Password: "x",
	})
	// We need an active connection - use sqlite but with sqlserver config
	// Actually the handler checks config.DBType before running query. So we need
	// GetConnection to return sqlserver. And GetActiveConnection to return non-nil.
	// So we need a real db for AddActiveConnection, and a config with sqlserver.
	db, _ := sql.Open("sqlite", ":memory:")
	store.AddActiveConnection("conn-sqlserver", db)
	defer store.RemoveActiveConnection("conn-sqlserver")

	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-sqlserver/explain", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-sqlserver")
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["success"] == true {
		t.Error("expected success false for sqlserver")
	}
}

func TestExplainHandler_SqliteExplain(t *testing.T) {
	connID, cleanup := setupExplainTest(t)
	defer cleanup()

	// SQLite uses EXPLAIN QUERY PLAN
	body, _ := json.Marshal(QueryRequest{Query: "SELECT * FROM test"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)

	// SQLite is not postgres/mysql/sqlserver - will hit "Unsupported database type"
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
}
