package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"

	"github.com/DATA-DOG/go-sqlmock"
)

type tableSizesResponse struct {
	Success   bool              `json:"success"`
	Sizes     []store.TableSize `json:"sizes"`
	UpdatedAt string            `json:"updated_at"`
	Error     string            `json:"error"`
}

type tableSizesStatusResponse struct {
	Success   bool   `json:"success"`
	Status    string `json:"status"`
	Count     int    `json:"count"`
	UpdatedAt string `json:"updated_at"`
	Error     string `json:"error"`
}

func TestGetTableSizesHandler_ReturnsCached(t *testing.T) {
	connID := "conn-table-sizes"

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	store.SetTableSizes(connID, []store.TableSize{
		{Schema: "public", Table: "users", SizeBytes: 2048},
		{Schema: "public", Table: "orders", SizeBytes: 4096},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/table-sizes", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()

	GetTableSizesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp tableSizesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error=%s", resp.Error)
	}
	if len(resp.Sizes) != 2 {
		t.Fatalf("expected 2 sizes, got %d", len(resp.Sizes))
	}
	if resp.UpdatedAt == "" {
		t.Fatalf("expected updated_at to be set")
	}
	if resp.Sizes[0].SizeBytes < resp.Sizes[1].SizeBytes {
		t.Fatalf("expected sizes to be sorted descending")
	}
}

func TestRefreshTableSizesHandler_Postgres(t *testing.T) {
	connID := "conn-table-refresh"

	restore := store.SetConnectionsFileForTest(filepath.Join(t.TempDir(), "connections.json"))
	t.Cleanup(restore)

	if err := store.WriteConnection(store.ConnectionConfig{
		ID:             connID,
		ConnectionName: "Test PG",
		DBType:         "postgres",
		DBName:         "app",
		User:           "user",
	}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	mock.ExpectQuery("relpages").
		WillReturnRows(sqlmock.NewRows([]string{"schema_name", "table_name", "size_bytes"}).
			AddRow("public", "users", int64(2048)))
	mock.ExpectQuery("SELECT current_schema\\(\\)").
		WillReturnRows(sqlmock.NewRows([]string{"current_schema"}).AddRow("public"))

	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/table-sizes/refresh", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()

	RefreshTableSizesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp tableSizesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error=%s", resp.Error)
	}
	if len(resp.Sizes) != 1 {
		t.Fatalf("expected 1 size, got %d", len(resp.Sizes))
	}
	if resp.Sizes[0].SizeBytes != 2048 {
		t.Fatalf("expected size 2048, got %d", resp.Sizes[0].SizeBytes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestGetTableSizesStatusHandler_ReturnsCount(t *testing.T) {
	connID := "conn-table-status"

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	store.SetTableSizes(connID, []store.TableSize{
		{Schema: "public", Table: "users", SizeBytes: 2048},
		{Schema: "public", Table: "orders", SizeBytes: 4096},
	})
	store.SetTableSizeStatus(connID, "ready", "")

	req := httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/table-sizes/status", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()

	GetTableSizesStatusHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp tableSizesStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error=%s", resp.Error)
	}
	if resp.Status != "ready" {
		t.Fatalf("expected status ready, got %s", resp.Status)
	}
	if resp.Count != 2 {
		t.Fatalf("expected count 2, got %d", resp.Count)
	}
	if resp.UpdatedAt == "" {
		t.Fatalf("expected updated_at to be set")
	}
}
