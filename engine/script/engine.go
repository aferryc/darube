package script

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"engine/db"
	"engine/store"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Provider resolves a connection ID to a callable handle.
// Implementations must not expose raw structs to JS; the ScriptEngine wraps handles into JS objects.
type Provider interface {
	Conn(ctx context.Context, id string) (ConnHandle, error)
}

// ConnHandle is the internal Go-side capability object.
// Methods that are not supported for a given connection kind should return ErrNotSupported.
type ConnHandle interface {
	Kind() string // "sql" or "redis"

	// SQL
	Query(ctx context.Context, sql string) ([]map[string]interface{}, error)
	Exec(ctx context.Context, sql string) (int64, error)
	One(ctx context.Context, sql string) (map[string]interface{}, error)
	Scalar(ctx context.Context, sql string) (interface{}, error)

	// Redis
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (*string, error) // nil means not found
	Del(ctx context.Context, key string) (int64, error)
}

var ErrNotSupported = errors.New("not supported for this connection type")

// ScriptEngine executes JavaScript scripts using goja.
// A new runtime is created per Run() call.
type ScriptEngine struct {
	provider Provider
}

func New(provider Provider) *ScriptEngine {
	return &ScriptEngine{provider: provider}
}

// NewDefault constructs a ScriptEngine that uses Darube's existing connection store.
// It will lazily connect saved connections if they are not currently active.
func NewDefault() *ScriptEngine {
	return New(DefaultProvider{})
}

// Run executes the given script in a fresh goja.Runtime.
// If ctx is canceled, the runtime is interrupted and the script throws.
func (e *ScriptEngine) Run(ctx context.Context, script string) (interface{}, error) {
	result, _, err := e.RunWithOutput(ctx, script)
	return result, err
}

// RunWithOutput is like Run, but also captures console output.
func (e *ScriptEngine) RunWithOutput(ctx context.Context, script string) (interface{}, []string, error) {
	vm := goja.New()

	// Context-driven interrupt hook for future timeouts/cancellation.
	// This interrupts JavaScript execution and makes RunString return an *goja.InterruptedError.
	if ctx != nil {
		go func() {
			<-ctx.Done()
			vm.Interrupt(ctx.Err())
		}()
	}

	var logs []string
	consoleObj := vm.NewObject()
	_ = consoleObj.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]interface{}, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, a.Export())
		}
		logs = append(logs, fmt.Sprint(parts...))
		return goja.Undefined()
	})
	_ = consoleObj.Set("error", func(call goja.FunctionCall) goja.Value {
		parts := make([]interface{}, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, a.Export())
		}
		logs = append(logs, fmt.Sprint(parts...))
		return goja.Undefined()
	})
	_ = vm.Set("console", consoleObj)

	// Utilities (no raw Go structs).
	sleepFn := func(call goja.FunctionCall) goja.Value {
		ms := call.Argument(0).ToInteger()
		if ms < 0 {
			ms = 0
		}
		d := time.Duration(ms) * time.Millisecond
		if ctx == nil {
			time.Sleep(d)
			return goja.Undefined()
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
			return goja.Undefined()
		case <-ctx.Done():
			panic(vm.NewGoError(ctx.Err()))
		}
	}

	utilsObj := vm.NewObject()
	_ = utilsObj.Set("sleep", sleepFn)
	_ = utilsObj.Set("uuidv7", func(goja.FunctionCall) goja.Value {
		u, err := uuid.NewV7()
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(u.String())
	})
	_ = utilsObj.Set("now", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().Format(time.RFC3339Nano))
	})
	_ = utilsObj.Set("nowUnixMs", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().UnixMilli())
	})
	_ = vm.Set("utils", utilsObj)

	// Convenience: expose sleep() directly.
	_ = vm.Set("sleep", sleepFn)

	// Inject global db API.
	dbObj := vm.NewObject()
	connCache := map[string]*goja.Object{}
	if err := dbObj.Set("conn", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		if id == "" || id == "undefined" || id == "null" {
			panic(vm.NewGoError(fmt.Errorf("db.conn(id): id is required")))
		}

		if cached := connCache[id]; cached != nil {
			return cached
		}

		handle, err := e.provider.Conn(ctx, id)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		obj := e.wrapConn(vm, ctx, id, handle)
		connCache[id] = obj
		return obj
	}); err != nil {
		return nil, logs, err
	}
	if err := vm.Set("db", dbObj); err != nil {
		return nil, logs, err
	}

	val, err := vm.RunString(script)
	if err != nil {
		return nil, logs, err
	}
	return val.Export(), logs, nil
}

func (e *ScriptEngine) wrapConn(vm *goja.Runtime, ctx context.Context, id string, handle ConnHandle) *goja.Object {
	obj := vm.NewObject()

	throw := func(err error) {
		panic(vm.NewGoError(err))
	}

	safe := func(fn func(goja.FunctionCall) goja.Value) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			defer func() {
				if r := recover(); r != nil {
					// Re-throw JS exceptions unchanged.
					if _, ok := r.(goja.Value); ok {
						panic(r)
					}
					panic(vm.NewGoError(fmt.Errorf("panic: %v", r)))
				}
			}()
			return fn(call)
		}
	}

	// SQL
	_ = obj.Set("query", safe(func(call goja.FunctionCall) goja.Value {
		sqlStr := call.Argument(0).String()
		rows, err := handle.Query(ctx, sqlStr)
		if err != nil {
			throw(err)
		}
		items := make([]interface{}, 0, len(rows))
		for _, r := range rows {
			items = append(items, toPlainObject(vm, r))
		}
		return vm.NewArray(items...)
	}))
	_ = obj.Set("exec", safe(func(call goja.FunctionCall) goja.Value {
		sqlStr := call.Argument(0).String()
		affected, err := handle.Exec(ctx, sqlStr)
		if err != nil {
			throw(err)
		}
		return vm.ToValue(affected)
	}))
	_ = obj.Set("one", safe(func(call goja.FunctionCall) goja.Value {
		sqlStr := call.Argument(0).String()
		row, err := handle.One(ctx, sqlStr)
		if err != nil {
			throw(err)
		}
		return toPlainObject(vm, row)
	}))
	_ = obj.Set("scalar", safe(func(call goja.FunctionCall) goja.Value {
		sqlStr := call.Argument(0).String()
		v, err := handle.Scalar(ctx, sqlStr)
		if err != nil {
			throw(err)
		}
		if v == nil {
			return goja.Null()
		}
		return vm.ToValue(v)
	}))

	// Redis
	_ = obj.Set("set", safe(func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		value := call.Argument(1).String()
		if err := handle.Set(ctx, key, value); err != nil {
			throw(err)
		}
		return goja.Undefined()
	}))
	_ = obj.Set("get", safe(func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		v, err := handle.Get(ctx, key)
		if err != nil {
			throw(err)
		}
		if v == nil {
			return goja.Null()
		}
		return vm.ToValue(*v)
	}))
	_ = obj.Set("del", safe(func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()
		n, err := handle.Del(ctx, key)
		if err != nil {
			throw(err)
		}
		return vm.ToValue(n)
	}))

	// Non-essential metadata (still plain values, no raw structs).
	_ = obj.Set("id", id)
	_ = obj.Set("kind", handle.Kind())

	return obj
}

func toPlainObject(vm *goja.Runtime, m map[string]interface{}) *goja.Object {
	o := vm.NewObject()
	for k, v := range m {
		// Values are expected to be primitives (string/number/bool/null) from the query normalization layer.
		_ = o.Set(k, v)
	}
	return o
}

// DefaultProvider resolves IDs from Darube's persisted connection stores.
// It connects lazily when an ID is not active.
type DefaultProvider struct{}

func (DefaultProvider) Conn(ctx context.Context, id string) (ConnHandle, error) {
	// Try SQL connection first.
	if cfg, err := store.GetConnection(id); err == nil && cfg != nil {
		if dbConn := store.GetActiveConnection(id); dbConn != nil {
			return &sqlHandle{id: id, db: dbConn}, nil
		}
		dbConn, err := db.Connect(*cfg)
		if err != nil {
			return nil, err
		}
		store.AddActiveConnection(id, dbConn)
		return &sqlHandle{id: id, db: dbConn}, nil
	}

	// Then Redis.
	if cfg, err := store.GetRedisConfig(id); err == nil && cfg != nil {
		if c := store.GetRedisConnection(id); c != nil {
			rc, ok := c.(*db.RedisClient)
			if !ok {
				return nil, fmt.Errorf("redis connection has unexpected type")
			}
			return &redisHandle{id: id, client: rc}, nil
		}
		rc, err := db.NewRedisClient(*cfg)
		if err != nil {
			return nil, err
		}
		store.AddRedisConnection(id, rc)
		return &redisHandle{id: id, client: rc}, nil
	}

	// Fallback: allow resolving by connection_name for user-friendly scripts.
	if resolved, ok, err := resolveSQLByName(id); err != nil {
		return nil, err
	} else if ok {
		return (DefaultProvider{}).Conn(ctx, resolved)
	}
	if resolved, ok, err := resolveRedisByName(id); err != nil {
		return nil, err
	} else if ok {
		return (DefaultProvider{}).Conn(ctx, resolved)
	}

	return nil, fmt.Errorf("connection '%s' not found", id)
}

func resolveSQLByName(name string) (string, bool, error) {
	conns, err := store.ReadConnections()
	if err != nil {
		return "", false, err
	}
	var match *store.ConnectionConfig
	for i := range conns {
		if conns[i].ConnectionName == name {
			if match != nil {
				return "", false, fmt.Errorf("multiple SQL connections share the name '%s' (use id instead)", name)
			}
			match = &conns[i]
		}
	}
	if match == nil {
		return "", false, nil
	}
	return match.ID, true, nil
}

func resolveRedisByName(name string) (string, bool, error) {
	conns, err := store.ReadRedisConnections()
	if err != nil {
		return "", false, err
	}
	var match *store.RedisConfig
	for i := range conns {
		if conns[i].ConnectionName == name {
			if match != nil {
				return "", false, fmt.Errorf("multiple Redis connections share the name '%s' (use id instead)", name)
			}
			match = &conns[i]
		}
	}
	if match == nil {
		return "", false, nil
	}
	return match.ID, true, nil
}

type sqlHandle struct {
	id string
	db *sql.DB
}

func (*sqlHandle) Kind() string { return "sql" }

func (h *sqlHandle) Query(ctx context.Context, q string) ([]map[string]interface{}, error) {
	if h.db == nil {
		return nil, fmt.Errorf("sql connection is not active")
	}
	rows, err := h.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		rowObj := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			rowObj[c] = normalizeSQLValue(values[i])
		}
		out = append(out, rowObj)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (h *sqlHandle) Exec(ctx context.Context, q string) (int64, error) {
	if h.db == nil {
		return 0, fmt.Errorf("sql connection is not active")
	}
	res, err := h.db.ExecContext(ctx, q)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func (h *sqlHandle) One(ctx context.Context, q string) (map[string]interface{}, error) {
	rows, err := h.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("expected 1 row, got 0")
	}
	if len(rows) > 1 {
		return nil, fmt.Errorf("expected 1 row, got %d", len(rows))
	}
	return rows[0], nil
}

func (h *sqlHandle) Scalar(ctx context.Context, q string) (interface{}, error) {
	if h.db == nil {
		return nil, fmt.Errorf("sql connection is not active")
	}
	rows, err := h.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("scalar: query returned no columns")
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("scalar: query returned no rows")
	}
	var v interface{}
	if err := rows.Scan(&v); err != nil {
		return nil, err
	}
	return normalizeSQLValue(v), nil
}

func (*sqlHandle) Set(context.Context, string, string) error { return ErrNotSupported }
func (*sqlHandle) Get(context.Context, string) (*string, error) {
	return nil, ErrNotSupported
}
func (*sqlHandle) Del(context.Context, string) (int64, error) { return 0, ErrNotSupported }

type redisHandle struct {
	id     string
	client *db.RedisClient
}

func (*redisHandle) Kind() string { return "redis" }

func (*redisHandle) Query(context.Context, string) ([]map[string]interface{}, error) {
	return nil, ErrNotSupported
}
func (*redisHandle) Exec(context.Context, string) (int64, error) { return 0, ErrNotSupported }
func (*redisHandle) One(context.Context, string) (map[string]interface{}, error) {
	return nil, ErrNotSupported
}
func (*redisHandle) Scalar(context.Context, string) (interface{}, error) { return nil, ErrNotSupported }

func (h *redisHandle) Set(ctx context.Context, key, value string) error {
	if h.client == nil || h.client.Client == nil {
		return fmt.Errorf("redis connection is not active")
	}
	return h.client.Client.Set(ctx, key, value, 0).Err()
}

func (h *redisHandle) Get(ctx context.Context, key string) (*string, error) {
	if h.client == nil || h.client.Client == nil {
		return nil, fmt.Errorf("redis connection is not active")
	}
	v, err := h.client.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (h *redisHandle) Del(ctx context.Context, key string) (int64, error) {
	if h.client == nil || h.client.Client == nil {
		return 0, fmt.Errorf("redis connection is not active")
	}
	return h.client.Client.Del(ctx, key).Result()
}

func normalizeSQLValue(v interface{}) interface{} {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339Nano)
	default:
		return t
	}
}
