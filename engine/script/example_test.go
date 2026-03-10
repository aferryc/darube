package script

import (
	"context"
	"fmt"
)

type fakeRedis struct{ mem map[string]string }

type fakeHandle struct {
	kind  string
	redis *fakeRedis
}

func (h fakeHandle) Kind() string { return h.kind }

// SQL
func (fakeHandle) Query(context.Context, string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{{"id": 1}, {"id": 2}}, nil
}
func (fakeHandle) Exec(context.Context, string) (int64, error) { return 0, ErrNotSupported }
func (fakeHandle) One(context.Context, string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": 1}, nil
}
func (fakeHandle) Scalar(context.Context, string) (interface{}, error) { return 1, nil }

// Redis
func (h fakeHandle) Set(_ context.Context, key, value string) error {
	if h.redis == nil {
		return ErrNotSupported
	}
	h.redis.mem[key] = value
	return nil
}
func (h fakeHandle) Get(_ context.Context, key string) (*string, error) {
	if h.redis == nil {
		return nil, ErrNotSupported
	}
	v, ok := h.redis.mem[key]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (h fakeHandle) Del(_ context.Context, key string) (int64, error) {
	if h.redis == nil {
		return 0, ErrNotSupported
	}
	_, ok := h.redis.mem[key]
	if !ok {
		return 0, nil
	}
	delete(h.redis.mem, key)
	return 1, nil
}

type fakeProvider struct{ redis *fakeRedis }

func (p fakeProvider) Conn(_ context.Context, id string) (ConnHandle, error) {
	switch id {
	case "prod-postgres":
		return fakeHandle{kind: "sql"}, nil
	case "cache":
		return fakeHandle{kind: "redis", redis: p.redis}, nil
	default:
		return nil, fmt.Errorf("connection '%s' not found", id)
	}
}

// ExampleScriptEngine_Run shows how to register db API and execute a script.
func ExampleScriptEngine_Run() {
	eng := New(fakeProvider{redis: &fakeRedis{mem: map[string]string{}}})

	script := `
const pg = db.conn("prod-postgres")
const redis = db.conn("cache")

const users = pg.query("SELECT id FROM users")
for (const u of users) {
  redis.set("user:" + u.id, "active")
}
`

	_, _ = eng.Run(context.Background(), script)
	// Output:
}
