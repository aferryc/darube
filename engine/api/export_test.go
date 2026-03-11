package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"engine/store"

	_ "modernc.org/sqlite"
)

func setupExportTest(t *testing.T) (string, string, func()) {
	dir := t.TempDir()
	connPath := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(connPath)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE export_test (id INTEGER, name TEXT)")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO export_test VALUES (1, 'a'), (2, 'b')")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	connID := "conn-export-test"
	store.WriteConnection(store.ConnectionConfig{
		ID:     connID,
		DBType: "sqlite",
		Host:   "localhost",
		Port:   0,
		User:   "",
		Password: "",
	})
	store.AddActiveConnection(connID, db)
	exportDir := filepath.Join(dir, "exports")
	os.MkdirAll(exportDir, 0755)
	return connID, exportDir, func() {
		store.RemoveActiveConnection(connID)
		restore()
	}
}

func TestExportHandler_MissingId(t *testing.T) {
	body := bytes.NewBufferString(`{"target_type":"data","target":"","format":"csv","headers":true,"destination_path":"/tmp","filename":"out","columns":["a"],"data":[[1]]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/export", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExportHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/connections/conn-1/export", nil)
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExportHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/export", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExportHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	params := ExportParams{
		TargetType:      "data",
		Format:          "csv",
		Headers:         true,
		DestinationPath: dir,
		Filename:        "out",
		Columns:         []string{"id"},
		Data:            [][]interface{}{{1}},
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestExportHandler_DataCSV(t *testing.T) {
	connID, exportDir, cleanup := setupExportTest(t)
	defer cleanup()

	params := ExportParams{
		TargetType:      "data",
		Format:          "csv",
		Headers:         true,
		DestinationPath: exportDir,
		Filename:        "data_export",
		Columns:         []string{"id", "name"},
		Data:            [][]interface{}{{1, "a"}, {2, "b"}},
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp ExportResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success: %+v", resp)
	}
	if resp.SavedTo == "" {
		t.Error("expected saved_to path")
	}
	if _, err := os.Stat(resp.SavedTo); os.IsNotExist(err) {
		t.Errorf("file not created: %s", resp.SavedTo)
	}
}

func TestExportHandler_DataJSON(t *testing.T) {
	connID, exportDir, cleanup := setupExportTest(t)
	defer cleanup()

	params := ExportParams{
		TargetType:      "data",
		Format:          "json",
		Headers:         true,
		DestinationPath: exportDir,
		Filename:        "data_export",
		Columns:         []string{"id", "name"},
		Data:            [][]interface{}{{1, "a"}, {2, "b"}},
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp ExportResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %+v", resp)
	}
	b, err := os.ReadFile(resp.SavedTo)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), `"id":1`) || !strings.Contains(string(b), `"name":"a"`) {
		t.Fatalf("unexpected json: %s", string(b))
	}
}

func TestExportHandler_DataSQL(t *testing.T) {
	connID, exportDir, cleanup := setupExportTest(t)
	defer cleanup()

	params := ExportParams{
		TargetType:      "data",
		Format:          "sql",
		DestinationPath: exportDir,
		Filename:        "data_export",
		Columns:         []string{"id", "name", "note"},
		Data:            [][]interface{}{{1, "a", nil}, {2, "b", "it's ok"}},
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp ExportResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %+v", resp)
	}
	b, err := os.ReadFile(resp.SavedTo)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "INSERT INTO exported_results") {
		t.Fatalf("expected default table name for non-table exports, got: %s", s)
	}
	if !strings.Contains(s, "NULL") {
		t.Fatalf("expected NULL, got: %s", s)
	}
	if !strings.Contains(s, "it''s ok") {
		t.Fatalf("expected escaped quote, got: %s", s)
	}
}

func TestExportHandler_TableJSON(t *testing.T) {
	connID, exportDir, cleanup := setupExportTest(t)
	defer cleanup()

	params := ExportParams{
		TargetType:      "table",
		Target:          "export_test",
		Format:          "json",
		DestinationPath: exportDir,
		Filename:        "table_export",
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp ExportResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Fatalf("expected success: %+v", resp)
	}
	if _, err := os.Stat(resp.SavedTo); err != nil {
		t.Fatalf("file: %v", err)
	}
}
