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

func setupQueryTest(t *testing.T) (string, func()) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// Create test table
	_, err = db.Exec("CREATE TABLE test (id INTEGER, name TEXT)")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO test VALUES (1, 'a'), (2, 'b')")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	connID := "conn-query-test"
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

func TestQueryHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/query", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestQueryHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/query", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestQueryHandler_EmptyQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/query", bytes.NewBufferString(`{"query":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestQueryHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/query", bytes.NewBufferString(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestQueryHandler_Select(t *testing.T) {
	connID, cleanup := setupQueryTest(t)
	defer cleanup()

	body, _ := json.Marshal(QueryRequest{Query: "SELECT * FROM test"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp QueryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success: %+v", resp)
	}
	if len(resp.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(resp.Columns))
	}
	if len(resp.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(resp.Rows))
	}
}

func TestQueryHandler_Mutation(t *testing.T) {
	connID, cleanup := setupQueryTest(t)
	defer cleanup()

	body, _ := json.Marshal(QueryRequest{Query: "INSERT INTO test VALUES (3, 'c')"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	QueryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp QueryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success: %+v", resp)
	}
	if resp.RowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", resp.RowsAffected)
	}
}
