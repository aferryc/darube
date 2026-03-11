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

func setupMutateTest(t *testing.T) (string, func()) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	connID := "conn-mutate-test"
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

func TestMutateDataHandler_MissingId(t *testing.T) {
	body := bytes.NewBufferString(`{"table":"users","mutations":[{"type":"insert","new_values":{"id":2,"name":"Bob"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/mutate", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestMutateDataHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/mutate", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestMutateDataHandler_EmptyTableOrMutations(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	tests := []struct {
		body string
	}{
		{`{"table":"","mutations":[{"type":"insert","new_values":{"id":1}}]}`},
		{`{"table":"users","mutations":[]}`},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/mutate", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "conn-1")
		rr := httptest.NewRecorder()
		MutateDataHandler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status: got %d for body %s", rr.Code, tc.body)
		}
	}
}

func TestMutateDataHandler_ConnectionNotActive(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	body := bytes.NewBufferString(`{"table":"users","mutations":[{"type":"insert","new_values":{"id":2,"name":"Bob"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/conn-1/mutate", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestMutateDataHandler_Insert(t *testing.T) {
	connID, cleanup := setupMutateTest(t)
	defer cleanup()

	reqBody := MutateRequest{
		Table: "users",
		Mutations: []MutationAction{
			{Type: "insert", NewValues: map[string]interface{}{"id": float64(2), "name": "Bob"}},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp MutateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success: %+v", resp)
	}
}

func TestMutateDataHandler_Update(t *testing.T) {
	connID, cleanup := setupMutateTest(t)
	defer cleanup()

	reqBody := MutateRequest{
		Table: "users",
		Mutations: []MutationAction{
			{
				Type:        "update",
				OriginalRow: map[string]interface{}{"id": float64(1), "name": "Alice"},
				NewValues:   map[string]interface{}{"name": "Alicia"},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestMutateDataHandler_Delete(t *testing.T) {
	connID, cleanup := setupMutateTest(t)
	defer cleanup()

	reqBody := MutateRequest{
		Table: "users",
		Mutations: []MutationAction{
			{
				Type:        "delete",
				OriginalRow: map[string]interface{}{"id": float64(1), "name": "Alice"},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	MutateDataHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestMutateDataHandler_ErrorsRollback(t *testing.T) {
	connID, cleanup := setupMutateTest(t)
	defer cleanup()

	tests := []MutateRequest{
		{
			Table: "users",
			Mutations: []MutationAction{
				{Type: "insert", NewValues: map[string]interface{}{}},
			},
		},
		{
			Table: "users",
			Mutations: []MutationAction{
				{Type: "update", OriginalRow: map[string]interface{}{"id": float64(999)}, NewValues: map[string]interface{}{"name": "x"}},
			},
		},
		{
			Table: "users",
			Mutations: []MutationAction{
				{Type: "delete", OriginalRow: map[string]interface{}{"id": float64(999)}},
			},
		},
		{
			Table: "users",
			Mutations: []MutationAction{
				{Type: "unknown", NewValues: map[string]interface{}{"id": float64(2)}},
			},
		},
	}

	for _, tc := range tests {
		body, _ := json.Marshal(tc)
		req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/mutate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", connID)
		rr := httptest.NewRecorder()
		MutateDataHandler(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func TestPlaceholderAndQuoteIdent(t *testing.T) {
	if placeholder("postgres", 3) != "$3" {
		t.Fatalf("postgres placeholder")
	}
	if placeholder("sqlserver", 2) != "@p2" {
		t.Fatalf("sqlserver placeholder")
	}
	if placeholder("sqlite", 1) != "?" {
		t.Fatalf("default placeholder")
	}

	if quoteIdent("mysql", "a`b") != "`a``b`" {
		t.Fatalf("mysql quote")
	}
	if quoteIdent("sqlserver", "a]b") != "[a]]b]" {
		t.Fatalf("sqlserver quote")
	}
	if quoteIdent("postgres", `a"b`) != `"a""b"` {
		t.Fatalf("postgres quote")
	}
}
