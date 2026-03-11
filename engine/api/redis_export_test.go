package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"engine/db"
	"engine/store"
)

func TestRedisToTabular_HashListScalar(t *testing.T) {
	cols, rows := redisToTabular("hash", map[string]interface{}{"b": 2, "a": "x"})
	if !reflect.DeepEqual(cols, []string{"Field", "Value"}) {
		t.Fatalf("unexpected cols: %#v", cols)
	}
	// keys sorted: a then b
	if len(rows) != 2 || rows[0][0] != "a" || rows[0][1] != "x" || rows[1][0] != "b" || rows[1][1] != "2" {
		t.Fatalf("unexpected rows: %#v", rows)
	}

	cols, rows = redisToTabular("list", []interface{}{"x", 1})
	if !reflect.DeepEqual(cols, []string{"Index", "Value"}) {
		t.Fatalf("unexpected cols: %#v", cols)
	}
	if len(rows) != 2 || rows[0][0] != 0 || rows[0][1] != "x" || rows[1][0] != 1 || rows[1][1] != "1" {
		t.Fatalf("unexpected rows: %#v", rows)
	}

	cols, rows = redisToTabular("string", map[string]interface{}{"nested": true})
	if !reflect.DeepEqual(cols, []string{"Value"}) || len(rows) != 1 || rows[0][0] == "" {
		t.Fatalf("unexpected scalar tabular: cols=%#v rows=%#v", cols, rows)
	}
}

func TestRedisExportHandler_Errors(t *testing.T) {
	store.ClearRedisConnectionsForTest()
	t.Cleanup(store.ClearRedisConnectionsForTest)

	// No active connection.
	req := httptest.NewRequest(http.MethodPost, "/api/redis/x/export", bytes.NewReader([]byte(`{}`)))
	req.SetPathValue("id", "x")
	rr := httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// Active connection but invalid JSON.
	store.AddRedisConnection("x", &db.RedisClient{Config: store.RedisConfig{ID: "x"}})
	t.Cleanup(func() { store.RemoveRedisConnection("x") })

	req = httptest.NewRequest(http.MethodPost, "/api/redis/x/export", bytes.NewReader([]byte(`{bad`)))
	req.SetPathValue("id", "x")
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid json: expected 400, got %d", rr.Code)
	}

	// Missing destination/filename.
	body, _ := json.Marshal(RedisExportParams{Format: "json"})
	req = httptest.NewRequest(http.MethodPost, "/api/redis/x/export", bytes.NewReader(body))
	req.SetPathValue("id", "x")
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing fields: expected 400, got %d", rr.Code)
	}

	// Unsupported format.
	body, _ = json.Marshal(RedisExportParams{Format: "nope", DestinationPath: t.TempDir(), Filename: "x"})
	req = httptest.NewRequest(http.MethodPost, "/api/redis/x/export", bytes.NewReader(body))
	req.SetPathValue("id", "x")
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unsupported: expected 400, got %d", rr.Code)
	}

	// Write error (destination does not exist).
	body, _ = json.Marshal(RedisExportParams{Format: "json", DestinationPath: filepath.Join(t.TempDir(), "missing"), Filename: "x"})
	req = httptest.NewRequest(http.MethodPost, "/api/redis/x/export", bytes.NewReader(body))
	req.SetPathValue("id", "x")
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("write error: expected 500, got %d", rr.Code)
	}
}
