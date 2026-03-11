package script

import (
	"context"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestDefaultProvider_SQLiteByIDAndCache(t *testing.T) {
	tmp := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(tmp, "connections.json"))
	defer restore()

	cfg := store.ConnectionConfig{
		ID:             "c1",
		ConnectionName: "local-sqlite",
		DBType:         "sqlite",
		FilePath:       ":memory:",
	}
	if err := store.WriteConnection(cfg); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	p := DefaultProvider{}
	h1, err := p.Conn(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Conn: %v", err)
	}
	if h1.Kind() != "sql" {
		t.Fatalf("expected sql kind, got %q", h1.Kind())
	}

	// Second call should reuse active connection.
	h2, err := p.Conn(context.Background(), "c1")
	if err != nil {
		t.Fatalf("Conn 2: %v", err)
	}
	if h2.Kind() != "sql" {
		t.Fatalf("expected sql kind, got %q", h2.Kind())
	}
}

func TestResolveSQLByName(t *testing.T) {
	tmp := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(tmp, "connections.json"))
	defer restore()

	_ = store.WriteConnection(store.ConnectionConfig{ID: "a", ConnectionName: "dup", DBType: "sqlite", FilePath: ":memory:"})
	_ = store.WriteConnection(store.ConnectionConfig{ID: "b", ConnectionName: "dup", DBType: "sqlite", FilePath: ":memory:"})

	_, ok, err := resolveSQLByName("missing")
	if err != nil || ok {
		t.Fatalf("expected missing, ok=%v err=%v", ok, err)
	}
	_, _, err = resolveSQLByName("dup")
	if err == nil {
		t.Fatalf("expected duplicate name error")
	}
}

func TestResolveRedisByName(t *testing.T) {
	tmp := t.TempDir()
	restore := store.SetRedisConnectionsFileForTest(filepath.Join(tmp, "redis.json"))
	defer restore()

	_ = store.WriteRedisConnection(store.RedisConfig{ID: "r1", ConnectionName: "cache", Host: "localhost", Port: 6379})

	id, ok, err := resolveRedisByName("cache")
	if err != nil || !ok || id != "r1" {
		t.Fatalf("id=%q ok=%v err=%v", id, ok, err)
	}
}
