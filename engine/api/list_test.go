package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"engine/store"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListConnectionsHandler(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()

	// Empty list
	req := httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	rr := httptest.NewRecorder()
	ListConnectionsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	var resp ListConnectionsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Connections) != 0 {
		t.Errorf("expected empty connections, got %d", len(resp.Connections))
	}

	// With saved connection
	cfg := store.ConnectionConfig{
		ID:             "c1",
		ConnectionName: "Test",
		DBType:         "postgres",
		Host:           "localhost",
		Port:           5432,
		User:           "u",
		Password:       "p",
	}
	if err := store.WriteConnection(cfg); err != nil {
		t.Fatal(err)
	}

	rr2 := httptest.NewRecorder()
	ListConnectionsHandler(rr2, httptest.NewRequest(http.MethodGet, "/api/connections", nil))
	if rr2.Code != http.StatusOK {
		t.Errorf("status: got %d", rr2.Code)
	}
	var resp2 ListConnectionsResponse
	if err := json.NewDecoder(rr2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp2.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(resp2.Connections))
	}
	if resp2.Connections[0].ID != "c1" || resp2.Connections[0].ConnectionName != "Test" {
		t.Errorf("unexpected connection: %+v", resp2.Connections[0])
	}
	if resp2.Connections[0].Status != "disconnected" {
		t.Errorf("expected disconnected, got %s", resp2.Connections[0].Status)
	}
}

func TestListConnectionsHandler_DoesNotPingOrConnect(t *testing.T) {
	restore := store.SetConnectionsFileForTest(filepath.Join(t.TempDir(), "connections.json"))
	defer restore()
	const id = "passive-status"
	if err := store.WriteConnection(store.ConnectionConfig{ID: id, DBType: "postgres", Host: "unavailable"}); err != nil {
		t.Fatal(err)
	}

	conn, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}

	checkStatus := func(want string) {
		t.Helper()
		rr := httptest.NewRecorder()
		ListConnectionsHandler(rr, httptest.NewRequest(http.MethodGet, "/api/connections", nil))
		var resp ListConnectionsResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		for _, c := range resp.Connections {
			if c.ID == id {
				if c.Status != want {
					t.Fatalf("status = %q, want %q", c.Status, want)
				}
				return
			}
		}
		t.Fatal("saved connection missing")
	}

	// A saved connection stays closed until explicitly opened.
	checkStatus("disconnected")
	store.AddActiveConnection(id, conn)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(id) })
	// No ping is expected: polling must not attempt network activity.
	for i := 0; i < 3; i++ {
		checkStatus("connected")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	mock.ExpectClose()
	if err := store.RemoveActiveConnection(id); err != nil {
		t.Fatal(err)
	}
	checkStatus("disconnected")
}

func TestListConnectionsHandler_MethodNotAllowed(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/connections", nil)
	rr := httptest.NewRecorder()
	ListConnectionsHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
}

func TestListConnectionsHandler_StoreError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := store.SetConnectionsFileForTest(path)
	defer restore()
	// Write corrupt JSON to trigger parse error
	os.WriteFile(path, []byte("not valid json"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	rr := httptest.NewRecorder()
	ListConnectionsHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}
