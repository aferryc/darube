package db

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

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
	f.kv[key] = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(redisValueString(value)), "\""), "\""))
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
	case "HGETALL":
		// Simulate go-redis reply shape: alternating key/value as []interface{}, sometimes []byte.
		cmd.SetVal([]interface{}{[]byte("f1"), []byte("v1"), "f2", "v2"})
	case "KEYS":
		cmd.SetVal([]interface{}{"k1", "k2"})
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
		return strings.TrimSpace(redisValueString(t))
	}
}

func redisValueString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return ""
	}
}

func stringSliceToInterface(s []string) []interface{} {
	out := make([]interface{}, 0, len(s))
	for _, v := range s {
		out = append(out, v)
	}
	return out
}

func TestRedisClient_Execute_TypedNilAndWarnings(t *testing.T) {
	cfg := store.RedisConfig{ID: "r1", ConnectionName: "cache"}
	c := &RedisClient{Client: newFakeRedis(), Config: cfg}

	// Missing GET -> typed nil, no error.
	res, err := c.Execute(context.Background(), "GET missing")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.DataType != "nil" || res.Value != nil {
		t.Fatalf("unexpected: %#v", res)
	}

	// SET and GET.
	if _, err := c.Execute(context.Background(), "SET k v"); err != nil {
		t.Fatalf("set: %v", err)
	}
	res, err = c.Execute(context.Background(), "GET k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if res.DataType != "string" || res.Value.(string) != "v" {
		t.Fatalf("unexpected: %#v", res)
	}

	// Dangerous command warning.
	res, err = c.Execute(context.Background(), "KEYS *")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if res.Message == "" {
		t.Fatalf("expected warning message")
	}
}

func TestRedisClient_NewFailsFastOnDialError(t *testing.T) {
	restoreDialer := SetRedisDialerForTest(func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New("dial fail")
	})
	defer restoreDialer()

	cfg := store.RedisConfig{Host: "127.0.0.1", Port: 6379, IsCluster: false}
	start := time.Now()
	_, err := NewRedisClient(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if time.Since(start) > 1*time.Second {
		t.Fatalf("expected failure quickly, took %s", time.Since(start))
	}
}

func TestRedisClient_Execute_JSONDetection(t *testing.T) {
	f := newFakeRedis()
	f.kv["j"] = `{"a":1}`
	c := &RedisClient{Client: f, Config: store.RedisConfig{ID: "r1"}}

	res, err := c.Execute(context.Background(), "GET j")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.DataType != "json" {
		t.Fatalf("expected json, got %#v", res)
	}
	m, ok := res.Value.(map[string]interface{})
	if !ok || m["a"] != float64(1) {
		t.Fatalf("unexpected value: %#v", res.Value)
	}
}

func TestRedisClient_Execute_HGETALL_NormalizesToMap(t *testing.T) {
	c := &RedisClient{Client: newFakeRedis(), Config: store.RedisConfig{ID: "r1"}}

	res, err := c.Execute(context.Background(), "HGETALL somehash")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.DataType != "hash" {
		t.Fatalf("expected hash, got %#v", res)
	}
	m, ok := res.Value.(map[string]interface{})
	if !ok || m["f1"] != "v1" || m["f2"] != "v2" {
		t.Fatalf("unexpected value: %#v", res.Value)
	}
}

func TestParseRedisCommandArgs_Errors(t *testing.T) {
	if _, err := parseRedisCommandArgs(`GET "unterminated`); err == nil {
		t.Fatalf("expected error for unterminated quote")
	}
	if _, err := parseRedisCommandArgs(`GET "bad\\`); err == nil {
		t.Fatalf("expected error for invalid escape")
	}
}
