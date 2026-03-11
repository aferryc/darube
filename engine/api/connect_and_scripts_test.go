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

	"github.com/DATA-DOG/go-sqlmock"
	_ "modernc.org/sqlite"
)

func setupConnectionsFile(t *testing.T) func() {
	t.Helper()
	path := filepath.Join(t.TempDir(), "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	return restore
}

func TestConnectNewHandler_SqliteValidationAndSuccess(t *testing.T) {
	restore := setupConnectionsFile(t)
	defer restore()

	// Method not allowed.
	req := httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	rr := httptest.NewRecorder()
	ConnectNewHandler(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}

	// Validation: missing file_path for sqlite.
	body, _ := json.Marshal(store.ConnectionConfig{ID: "c1", ConnectionName: "x", DBType: "sqlite"})
	req = httptest.NewRequest(http.MethodPost, "/api/connections", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	ConnectNewHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// Success: sqlite in-memory.
	cfg := store.ConnectionConfig{ID: "c2", ConnectionName: "mem", DBType: "sqlite", FilePath: ":memory:"}
	body, _ = json.Marshal(cfg)
	req = httptest.NewRequest(http.MethodPost, "/api/connections", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	ConnectNewHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp CommandOutput
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success || resp.ID != "c2" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
	if !store.IsConnected("c2") {
		t.Fatalf("expected connection to be active")
	}
	_ = store.RemoveActiveConnection("c2")
}

func TestConnectSavedHandler_SqliteNotFoundConnectAndAlreadyConnected(t *testing.T) {
	restore := setupConnectionsFile(t)
	defer restore()

	// Not found.
	body, _ := json.Marshal(ConnectSavedRequest{ID: "missing"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/connect", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ConnectSavedHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	// Save a sqlite config and connect.
	cfg := store.ConnectionConfig{ID: "c1", ConnectionName: "mem", DBType: "sqlite", FilePath: ":memory:"}
	if err := store.WriteConnection(cfg); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	body, _ = json.Marshal(ConnectSavedRequest{ID: "c1"})
	req = httptest.NewRequest(http.MethodPost, "/api/connections/connect", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	ConnectSavedHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !store.IsConnected("c1") {
		t.Fatalf("expected c1 to be connected")
	}

	// Already connected branch.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/connections/connect", bytes.NewReader(body))
	ConnectSavedHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	_ = store.RemoveActiveConnection("c1")
}

func TestRunScriptHandler_BasicAndErrors(t *testing.T) {
	// Method not allowed.
	req := httptest.NewRequest(http.MethodGet, "/api/scripts/run", nil)
	rr := httptest.NewRecorder()
	RunScriptHandler(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}

	// Invalid body.
	req = httptest.NewRequest(http.MethodPost, "/api/scripts/run", bytes.NewBufferString("nope"))
	rr = httptest.NewRecorder()
	RunScriptHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// Missing script.
	body, _ := json.Marshal(ScriptRunRequest{Script: ""})
	req = httptest.NewRequest(http.MethodPost, "/api/scripts/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	RunScriptHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// Success.
	body, _ = json.Marshal(ScriptRunRequest{Script: "1 + 1"})
	req = httptest.NewRequest(http.MethodPost, "/api/scripts/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	RunScriptHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp ScriptRunResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %#v", resp)
	}

	// Script error path.
	body, _ = json.Marshal(ScriptRunRequest{Script: `throw new Error("boom")`})
	req = httptest.NewRequest(http.MethodPost, "/api/scripts/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	RunScriptHandler(rr, req)
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Success || resp.Error == "" {
		t.Fatalf("expected error: %#v", resp)
	}
}

func TestGetMetadataEntitiesHandler_PostgresMock(t *testing.T) {
	restore := setupConnectionsFile(t)
	defer restore()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	mock.ExpectQuery("FROM information_schema\\.tables").WillReturnRows(
		sqlmock.NewRows([]string{"table_schema", "table_name", "table_type", "column_name", "data_type"}).
			AddRow("public", "t", "BASE TABLE", "id", "int"),
	)
	mock.ExpectQuery("FROM pg_indexes").WillReturnRows(
		sqlmock.NewRows([]string{"schemaname", "tablename", "indexname"}).
			AddRow("public", "t", "t_pkey"),
	)

	connID := "meta-entities"
	if err := store.WriteConnection(store.ConnectionConfig{ID: connID, ConnectionName: "m", DBType: "postgres"}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}
	store.AddActiveConnection(connID, conn)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	req := httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/entities", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	GetMetadataEntitiesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}

	// Also cover missing id.
	req = httptest.NewRequest(http.MethodGet, "/api/connections//metadata/entities", nil)
	rr = httptest.NewRecorder()
	GetMetadataEntitiesHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing id, got %d", rr.Code)
	}

	// Cover not active.
	req = httptest.NewRequest(http.MethodGet, "/api/connections/x/metadata/entities", nil)
	req.SetPathValue("id", "x")
	rr = httptest.NewRecorder()
	GetMetadataEntitiesHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for inactive, got %d", rr.Code)
	}
}
