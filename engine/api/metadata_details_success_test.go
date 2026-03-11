package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"engine/store"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMetadataDetailsHandlers_PostgresMockSuccess(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "connections.json"))
	defer restore()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	connID := "meta-details"
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	if err := store.WriteConnection(store.ConnectionConfig{ID: connID, DBType: "postgres", ConnectionName: "m"}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.columns")).
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type", "character_maximum_length", "is_nullable", "column_default"}).
			AddRow("id", "int8", nil, "NO", nil).
			AddRow("name", "text", nil, "YES", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/schemas/public/tables/users/dml", nil)
	req.SetPathValue("id", connID)
	req.SetPathValue("schema", "public")
	req.SetPathValue("table", "users")
	rr := httptest.NewRecorder()
	GetMetadataTableDMLHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dml: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var dmlResp DMLResponse
	_ = json.NewDecoder(rr.Body).Decode(&dmlResp)
	if !dmlResp.Success || !strings.Contains(dmlResp.DML, "CREATE TABLE") {
		t.Fatalf("unexpected dml resp: %#v", dmlResp)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM\n\t\t\tpg_class t")).
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "is_unique", "is_primary"}).
			AddRow("users_pkey", "id", true, true))

	req = httptest.NewRequest(http.MethodGet, "/api/connections/"+connID+"/metadata/schemas/public/tables/users/indexes", nil)
	req.SetPathValue("id", connID)
	req.SetPathValue("schema", "public")
	req.SetPathValue("table", "users")
	rr = httptest.NewRecorder()
	GetMetadataTableIndexesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("indexes: expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var idxResp IndexesResponse
	_ = json.NewDecoder(rr.Body).Decode(&idxResp)
	if !idxResp.Success || len(idxResp.Indexes) != 1 || idxResp.Indexes[0].Name != "users_pkey" {
		t.Fatalf("unexpected indexes resp: %#v", idxResp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

