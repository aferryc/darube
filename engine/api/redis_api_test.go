package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"engine/db"
	"engine/store"

	"github.com/redis/go-redis/v9"
)

type fakeRedis struct {
	mu sync.Mutex
	kv map[string]string
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{kv: map[string]string{}}
}

func (f *fakeRedis) Close() error { return nil }

func (f *fakeRedis) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "PING")
	cmd.SetVal("PONG")
	return cmd
}

func (f *fakeRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	f.mu.Lock()
	f.kv[key] = redisArgString(value)
	f.mu.Unlock()

	cmd := redis.NewStatusCmd(ctx, "SET", key, value)
	cmd.SetVal("OK")
	return cmd
}

func (f *fakeRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	f.mu.Lock()
	v, ok := f.kv[key]
	f.mu.Unlock()

	cmd := redis.NewStringCmd(ctx, "GET", key)
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(v)
	return cmd
}

func (f *fakeRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	var deleted int64
	f.mu.Lock()
	for _, k := range keys {
		if _, ok := f.kv[k]; ok {
			delete(f.kv, k)
			deleted++
		}
	}
	f.mu.Unlock()

	cmd := redis.NewIntCmd(ctx, append([]interface{}{"DEL"}, stringSliceToInterface(keys)...)...)
	cmd.SetVal(deleted)
	return cmd
}

func (f *fakeRedis) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	cmd := redis.NewCmd(ctx, args...)
	if len(args) == 0 {
		cmd.SetErr(errors.New("missing command"))
		return cmd
	}
	name := strings.ToUpper(redisArgString(args[0]))
	switch name {
	case "PING":
		cmd.SetVal("PONG")
	case "SET":
		if len(args) < 3 {
			cmd.SetErr(errors.New("SET requires key and value"))
			return cmd
		}
		key := redisArgString(args[1])
		val := redisArgString(args[2])
		f.mu.Lock()
		f.kv[key] = val
		f.mu.Unlock()
		cmd.SetVal("OK")
	case "GET":
		if len(args) < 2 {
			cmd.SetErr(errors.New("GET requires key"))
			return cmd
		}
		key := redisArgString(args[1])
		f.mu.Lock()
		val, ok := f.kv[key]
		f.mu.Unlock()
		if !ok {
			cmd.SetErr(redis.Nil)
			return cmd
		}
		cmd.SetVal(val)
	default:
		cmd.SetErr(errors.New("unsupported command"))
	}
	return cmd
}

func redisArgString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func stringSliceToInterface(s []string) []interface{} {
	out := make([]interface{}, 0, len(s))
	for _, v := range s {
		out = append(out, v)
	}
	return out
}

func setupRedisApiTest(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "redis_connections.json")
	restoreStore := store.SetRedisConnectionsFileForTest(path)

	prevNewRedisClient := newRedisClient
	newRedisClient = func(config store.RedisConfig) (*db.RedisClient, error) {
		return &db.RedisClient{Client: newFakeRedis(), Config: config}, nil
	}

	return func() {
		newRedisClient = prevNewRedisClient
		restoreStore()
		store.ClearRedisConnectionsForTest()
	}
}

func TestTestRedisHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	cfg := store.RedisConfig{
		Host: "127.0.0.1",
		Port: 6379,
	}

	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/redis/test", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	TestRedisHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp CommandOutput
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Errorf("expected success, got error: %s", resp.Error)
	}
}

func TestConnectRedisHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	cfg := store.RedisConfig{
		ID:             "redis-1",
		ConnectionName: "test-redis",
		Host:           "127.0.0.1",
		Port:           6379,
	}

	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/redis", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ConnectRedisHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	if !store.IsRedisConnected("redis-1") {
		t.Error("expected redis-1 to be connected")
	}
}

func TestRedisQueryHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	// 1. Connect first
	connID := "q-test"
	c, _ := newRedisClient(store.RedisConfig{ID: connID, Host: "127.0.0.1", Port: 6379})
	store.AddRedisConnection(connID, c)

	// 2. Query
	queryBody, _ := json.Marshal(RedisQueryRequest{Command: "SET foo bar"})
	req := httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/query", bytes.NewReader(queryBody))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	RedisQueryHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp RedisQueryResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success {
		t.Errorf("query failed: %s", resp.Error)
	}
}

func TestDeleteRedisConnectionHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "del-test"
	store.WriteRedisConnection(store.RedisConfig{ID: connID, Host: "localhost"})

	req := httptest.NewRequest(http.MethodDelete, "/api/redis/"+connID, nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	DeleteRedisConnectionHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	
	conns, _ := store.ReadRedisConnections()
	for _, c := range conns {
		if c.ID == connID {
			t.Error("expected connection to be deleted from disk")
		}
	}
}

func TestDisconnectRedisHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "disc-test"
	c, _ := newRedisClient(store.RedisConfig{ID: connID, Host: "127.0.0.1", Port: 6379})
	store.AddRedisConnection(connID, c)

	req := httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/disconnect", nil)
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	DisconnectRedisHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	
	if store.IsRedisConnected(connID) {
		t.Error("expected connection to be disconnected")
	}
}

func TestPatchRedisFolderHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "patch-test"
	store.WriteRedisConnection(store.RedisConfig{ID: connID, Host: "localhost"})

	body, _ := json.Marshal(map[string]string{"folder_id": "f1"})
	req := httptest.NewRequest(http.MethodPatch, "/api/redis/"+connID+"/folder", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	PatchRedisFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	
	cfg, _ := store.GetRedisConfig(connID)
	if cfg.FolderID != "f1" {
		t.Errorf("expected folder_id f1, got %s", cfg.FolderID)
	}
}

func TestConnectSavedRedisHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "reconnect-test"
	store.WriteRedisConnection(store.RedisConfig{ID: connID, Host: "localhost"})

	body, _ := json.Marshal(map[string]string{"id": connID})
	req := httptest.NewRequest(http.MethodPost, "/api/redis/reconnect", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	ConnectSavedRedisHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRedisExportHandler(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "export-test"
	store.AddRedisConnection(connID, &db.RedisClient{Client: newFakeRedis(), Config: store.RedisConfig{ID: connID}})

	params := RedisExportParams{
		Format:          "json",
		DestinationPath: t.TempDir(),
		Filename:        "export_test",
		DataType:        "string",
		Value:           "v1",
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/export", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	RedisExportHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
	}
	
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected JSON response, got %s", rr.Header().Get("Content-Type"))
	}

	// Test CSV
	params.Format = "csv"
	body, _ = json.Marshal(params)
	req = httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/export", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("csv: expected 200, got %d", rr.Code)
	}

	// Test Excel
	params.Format = "excel"
	body, _ = json.Marshal(params)
	req = httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/export", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("excel: expected 200, got %d", rr.Code)
	}

	// Test Hash Tabular
	params.Format = "json"
	params.DataType = "hash"
	params.Value = map[string]interface{}{"f1": "v1"}
	body, _ = json.Marshal(params)
	req = httptest.NewRequest(http.MethodPost, "/api/redis/"+connID+"/export", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr = httptest.NewRecorder()
	RedisExportHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("hash: expected 200, got %d", rr.Code)
	}
}

func TestConnectRedisHandler_PreservesFolderIDOnUpdate(t *testing.T) {
	cleanup := setupRedisApiTest(t)
	defer cleanup()

	connID := "redis-folder"
	if err := store.WriteRedisConnection(store.RedisConfig{ID: connID, Host: "localhost", FolderID: "f1"}); err != nil {
		t.Fatalf("WriteRedisConnection: %v", err)
	}

	cfg := store.RedisConfig{
		ConnectionName: "updated",
		Host:           "127.0.0.1",
		Port:           6379,
		FolderID:       "",
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPut, "/api/redis/"+connID, bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()
	ConnectRedisHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	updated, err := store.GetRedisConfig(connID)
	if err != nil || updated == nil {
		t.Fatalf("GetRedisConfig: %v", err)
	}
	if updated.FolderID != "f1" {
		t.Fatalf("expected folder_id preserved, got %q", updated.FolderID)
	}
}
