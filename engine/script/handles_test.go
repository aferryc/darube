package script

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNormalizeSQLValue(t *testing.T) {
	if normalizeSQLValue(nil) != nil {
		t.Fatalf("expected nil")
	}
	if normalizeSQLValue([]byte("x")).(string) != "x" {
		t.Fatalf("expected []byte -> string")
	}
}

func TestSQLHandle_Sqlite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER, name TEXT); INSERT INTO t VALUES (1,'a'),(2,'b');`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	h := &sqlHandle{id: "x", db: db}
	rows, err := h.Query(context.Background(), "SELECT id, name FROM t ORDER BY id")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 2 || rows[0]["id"] == nil || rows[0]["name"] != "a" {
		t.Fatalf("unexpected rows: %#v", rows)
	}

	aff, err := h.Exec(context.Background(), "UPDATE t SET name='c' WHERE id=1")
	if err != nil || aff != 1 {
		t.Fatalf("Exec: aff=%d err=%v", aff, err)
	}

	one, err := h.One(context.Background(), "SELECT id FROM t WHERE id=2")
	if err != nil || one["id"] == nil {
		t.Fatalf("One: %#v err=%v", one, err)
	}

	v, err := h.Scalar(context.Background(), "SELECT name FROM t WHERE id=2")
	if err != nil || v.(string) != "b" {
		t.Fatalf("Scalar: %#v err=%v", v, err)
	}
}

func TestSQLHandle_Errors(t *testing.T) {
	h := &sqlHandle{id: "x", db: nil}
	if _, err := h.Query(context.Background(), "select 1"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := h.Exec(context.Background(), "select 1"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := h.Scalar(context.Background(), "select 1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSQLHandle_RedisMethodsNotSupported(t *testing.T) {
	h := &sqlHandle{id: "x", db: nil}
	if err := h.Set(context.Background(), "k", "v"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
	if _, err := h.Get(context.Background(), "k"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
	if _, err := h.Del(context.Background(), "k"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
}

func TestRedisHandle_Inactive(t *testing.T) {
	h := &redisHandle{id: "r", client: nil}
	if err := h.Set(context.Background(), "k", "v"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := h.Get(context.Background(), "k"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := h.Del(context.Background(), "k"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRedisHandle_SQLMethodsNotSupported(t *testing.T) {
	h := &redisHandle{id: "r", client: nil}
	if h.Kind() != "redis" {
		t.Fatalf("expected redis kind")
	}
	if _, err := h.Query(context.Background(), "select 1"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
	if _, err := h.Exec(context.Background(), "select 1"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
	if _, err := h.One(context.Background(), "select 1"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
	if _, err := h.Scalar(context.Background(), "select 1"); err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported")
	}
}
