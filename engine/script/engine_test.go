package script

import (
	"context"
	"strings"
	"testing"
)

type providerMap map[string]ConnHandle

func (p providerMap) Conn(_ context.Context, id string) (ConnHandle, error) {
	h, ok := p[id]
	if !ok {
		return nil, context.Canceled
	}
	return h, nil
}

type handleSQL struct{}

func (handleSQL) Kind() string { return "sql" }
func (handleSQL) Query(context.Context, string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{{"id": 1, "name": "a"}}, nil
}
func (handleSQL) Exec(context.Context, string) (int64, error) { return 2, nil }
func (handleSQL) One(context.Context, string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": 7}, nil
}
func (handleSQL) Scalar(context.Context, string) (interface{}, error) { return nil, nil }
func (handleSQL) Set(context.Context, string, string) error           { return ErrNotSupported }
func (handleSQL) Get(context.Context, string) (*string, error)        { return nil, ErrNotSupported }
func (handleSQL) Del(context.Context, string) (int64, error)          { return 0, ErrNotSupported }

type handleRedis struct{ mem map[string]string }

func (handleRedis) Kind() string { return "redis" }
func (handleRedis) Query(context.Context, string) ([]map[string]interface{}, error) {
	return nil, ErrNotSupported
}
func (handleRedis) Exec(context.Context, string) (int64, error) { return 0, ErrNotSupported }
func (handleRedis) One(context.Context, string) (map[string]interface{}, error) {
	return nil, ErrNotSupported
}
func (handleRedis) Scalar(context.Context, string) (interface{}, error) { return nil, ErrNotSupported }
func (h handleRedis) Set(_ context.Context, key, value string) error {
	h.mem[key] = value
	return nil
}
func (h handleRedis) Get(_ context.Context, key string) (*string, error) {
	v, ok := h.mem[key]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (h handleRedis) Del(_ context.Context, key string) (int64, error) {
	if _, ok := h.mem[key]; !ok {
		return 0, nil
	}
	delete(h.mem, key)
	return 1, nil
}

func TestScriptEngine_DBConnCachingAndErrors(t *testing.T) {
	eng := New(providerMap{
		"pg":    handleSQL{},
		"redis": handleRedis{mem: map[string]string{}},
	})

	_, _, err := eng.RunWithOutput(context.Background(), `db.conn("missing")`)
	if err == nil {
		t.Fatalf("expected error")
	}

	out, _, err := eng.RunWithOutput(context.Background(), `
const a = db.conn("pg")
const b = db.conn("pg")
a === b
`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != true {
		t.Fatalf("expected true, got %#v", out)
	}
}

func TestScriptEngine_MethodsAndNulls(t *testing.T) {
	mem := map[string]string{}
	eng := New(providerMap{
		"pg":    handleSQL{},
		"redis": handleRedis{mem: mem},
	})

	out, logs, err := eng.RunWithOutput(context.Background(), `
console.log("x", 1)
const pg = db.conn("pg")
const redis = db.conn("redis")

const q = pg.query("select")
const n = pg.exec("update")
const one = pg.one("one")
const s = pg.scalar("scalar") // nil -> null

redis.set("k","v")
const got = redis.get("missing") // null
const got2 = redis.get("k")
const del0 = redis.del("missing")
const del1 = redis.del("k")

var out = { q: q, n: n, one: one, s: s, got: got, got2: got2, del0: del0, del1: del1 }
out
`)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	m := out.(map[string]interface{})
	if m["n"].(int64) != 2 {
		t.Fatalf("exec: %#v", m["n"])
	}
	if m["s"] != nil {
		t.Fatalf("scalar nil should export as nil, got %#v", m["s"])
	}
	if m["got"] != nil {
		t.Fatalf("missing redis key should export nil, got %#v", m["got"])
	}
	if m["got2"].(string) != "v" {
		t.Fatalf("got2: %#v", m["got2"])
	}
	if len(logs) == 0 || !strings.Contains(logs[0], "x1") {
		t.Fatalf("logs: %#v", logs)
	}
}
