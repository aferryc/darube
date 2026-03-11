package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"testing"

	"engine/store"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestExplainHandler_PostgresAndMySQLOk(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "connections.json"))
	defer restore()

	// Postgres success.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	connID := "explain-pg"
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })
	_ = store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "postgres"})

	mock.ExpectQuery(regexp.QuoteMeta("EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON)")).
		WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(`[{"Plan":{"Node Type":"Seq Scan"}}]`))

	body, _ := json.Marshal(QueryRequest{Query: "select 1"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("postgres: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	// MySQL success.
	db2, mock2, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	connID = "explain-mysql"
	store.AddActiveConnection(connID, db2)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })
	_ = store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "mysql"})

	mock2.ExpectQuery(regexp.QuoteMeta("EXPLAIN FORMAT=JSON")).
		WillReturnRows(sqlmock.NewRows([]string{"EXPLAIN"}).AddRow(`{"query_block":{}}`))

	req = httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("mysql: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("pg expectations: %v", err)
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("mysql expectations: %v", err)
	}
}

func TestExplainHandler_NotImplementedBranches(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "connections.json"))
	defer restore()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	connID := "explain-sqlserver"
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })
	_ = store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "sqlserver"})

	body, _ := json.Marshal(QueryRequest{Query: "select 1"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rr.Code)
	}

	db2, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	connID = "explain-unsupported"
	store.AddActiveConnection(connID, db2)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })
	_ = store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "sqlite"})
	req = httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rr.Code)
	}
}

func TestExplainHandler_ErrorBranches(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "connections.json"))
	defer restore()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	connID := "explain-errors"
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	body, _ := json.Marshal(QueryRequest{Query: "select 1"})

	// Config not found.
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("config missing: expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	// No plan returned.
	_ = store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "postgres"})
	mock.ExpectQuery(regexp.QuoteMeta("EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON)")).
		WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}))
	req = httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("no plan: expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	// Invalid JSON plan.
	mock.ExpectQuery(regexp.QuoteMeta("EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON)")).
		WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow(`not json`))
	req = httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/explain", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	ExplainHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("bad json: expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
