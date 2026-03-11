package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRedisConfig_FileCRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "redis.json")
	restore := SetRedisConnectionsFileForTest(path)
	defer restore()

	// Read when missing -> empty list
	conns, err := ReadRedisConnections()
	if err != nil || len(conns) != 0 {
		t.Fatalf("ReadRedisConnections: %v %#v", err, conns)
	}

	cfg := RedisConfig{ID: "r1", ConnectionName: "cache", Host: "localhost", Port: 6379, IsCluster: true, FolderID: "f1"}
	if err := WriteRedisConnection(cfg); err != nil {
		t.Fatalf("WriteRedisConnection: %v", err)
	}

	got, err := GetRedisConfig("r1")
	if err != nil || got == nil || got.ConnectionName != "cache" || got.FolderID != "f1" {
		t.Fatalf("GetRedisConfig: got=%#v err=%v", got, err)
	}

	// Update existing
	cfg.ConnectionName = "cache2"
	if err := WriteRedisConnection(cfg); err != nil {
		t.Fatalf("WriteRedisConnection update: %v", err)
	}
	got, _ = GetRedisConfig("r1")
	if got.ConnectionName != "cache2" {
		t.Fatalf("expected updated name, got %#v", got)
	}

	if err := DeleteRedisConnection("r1"); err != nil {
		t.Fatalf("DeleteRedisConnection: %v", err)
	}
	_, err = GetRedisConfig("r1")
	if err == nil {
		t.Fatalf("expected not found")
	}
}

func TestRedisConfig_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "redis.json")
	restore := SetRedisConnectionsFileForTest(path)
	defer restore()

	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := ReadRedisConnections()
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
