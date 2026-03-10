package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"engine/store"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	UniversalClient redis.UniversalClient
	Config         store.RedisConfig
}

type RedisResult struct {
	DataType string      `json:"data_type"`
	Value    interface{} `json:"value"`
	Message  string      `json:"message,omitempty"`
}

func NewRedisClient(config store.RedisConfig) (*RedisClient, error) {
	var client redis.UniversalClient
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	if config.IsCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{addr},
			Username: config.User,
			Password: config.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: config.User,
			Password: config.Password,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		UniversalClient: client,
		Config:          config,
	}, nil
}

func (c *RedisClient) Close() error {
	return c.UniversalClient.Close()
}

func (c *RedisClient) Execute(ctx context.Context, commandStr string) (*RedisResult, error) {
	parts, err := parseRedisCommandArgs(commandStr)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmdName := strings.ToUpper(parts[0])
	
	// Performance Warnings check (handled in API/Frontend ideally, but logic here helps)
	isDangerous := false
	if cmdName == "KEYS" || cmdName == "FLUSHALL" || cmdName == "FLUSHDB" {
		isDangerous = true
	}
	// For SCAN, it's safer than KEYS but still noted in plan.

	args := make([]interface{}, len(parts)-1)
	for i, p := range parts[1:] {
		args[i] = p
	}

	cmd := c.UniversalClient.Do(ctx, append([]interface{}{parts[0]}, args...)...)
	val, err := cmd.Result()
	if err != nil {
		// go-redis uses redis.Nil to indicate "not found" (e.g. GET missing-key).
		// That's not an execution failure; surface it as a typed nil result.
		if errors.Is(err, redis.Nil) {
			result := &RedisResult{
				DataType: "nil",
				Value:    nil,
			}
			if isDangerous {
				result.Message = "Warning: This command may impact Redis performance."
			}
			return result, nil
		}
		return nil, err
	}

	result := &RedisResult{
		Value: normalizeRedisValue(cmdName, val),
	}

	if isDangerous {
		result.Message = "Warning: This command may impact Redis performance."
	}

	if val == nil {
		result.DataType = "nil"
		return result, nil
	}

	// Determine data type based on command or value
	// This is a bit simplified, ideally we'd use TYPE command if we queried a key
	switch v := result.Value.(type) {
	case string:
		result.DataType = "string"
		// Try to see if it's JSON
		var js interface{}
		if err := json.Unmarshal([]byte(v), &js); err == nil {
			result.DataType = "json"
			result.Value = js
		}
	case []interface{}:
		result.DataType = "list"
		// If it has even number of elements and looks like a hash (from HGETALL)
		// we can tentatively label it, but "list" is safer.
	case map[string]interface{}:
		result.DataType = "hash"
	case map[string]string:
		result.DataType = "hash"
	case int64:
		result.DataType = "integer"
	case float64:
		result.DataType = "number"
	case bool:
		result.DataType = "boolean"
	case nil:
		result.DataType = "nil"
	default:
		result.DataType = fmt.Sprintf("%T", v)
	}

	return result, nil
}

// parseRedisCommandArgs splits a command string into tokens, supporting quotes and escapes.
// Examples:
//   SET k "hello world"
//   HSET k f "{\"a\":1}\n"
func parseRedisCommandArgs(s string) ([]string, error) {
	var out []string
	var cur []rune
	var quote rune
	escaped := false

	flush := func() {
		if len(cur) == 0 {
			return
		}
		out = append(out, string(cur))
		cur = cur[:0]
	}

	for _, r := range []rune(strings.TrimSpace(s)) {
		if escaped {
			switch r {
			case 'n':
				cur = append(cur, '\n')
			case 'r':
				cur = append(cur, '\r')
			case 't':
				cur = append(cur, '\t')
			case '\\', '"', '\'':
				cur = append(cur, r)
			default:
				// Keep unknown escapes as-is.
				cur = append(cur, r)
			}
			escaped = false
			continue
		}

		if quote != 0 {
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
				continue
			}
			cur = append(cur, r)
			continue
		}

		switch r {
		case '"', '\'':
			quote = r
		case ' ', '\n', '\r', '\t':
			flush()
		default:
			cur = append(cur, r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("invalid escape at end of command")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in command")
	}
	flush()
	return out, nil
}

func normalizeRedisValue(cmdName string, val interface{}) interface{} {
	if val == nil {
		return nil
	}

	// Special-case common commands to provide nicer (and safer) shapes.
	// go-redis returns HGETALL via Do() as []interface{} alternating key/value, often as []byte.
	if cmdName == "HGETALL" {
		if list, ok := val.([]interface{}); ok && len(list)%2 == 0 {
			m := make(map[string]interface{}, len(list)/2)
			for i := 0; i < len(list); i += 2 {
				k := normalizeScalarToString(list[i])
				m[k] = normalizeRedisValue("", list[i+1])
			}
			return m
		}
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int64, float64, bool:
		return v
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, it := range v {
			out = append(out, normalizeRedisValue("", it))
		}
		return out
	case []string:
		out := make([]interface{}, 0, len(v))
		for _, it := range v {
			out = append(out, it)
		}
		return out
	case map[string]string:
		out := make(map[string]interface{}, len(v))
		for k, vv := range v {
			out[k] = vv
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, vv := range v {
			out[k] = normalizeRedisValue("", vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, vv := range v {
			out[normalizeScalarToString(k)] = normalizeRedisValue("", vv)
		}
		return out
	default:
		// Last resort: stringify to avoid "unsupported type" marshal failures.
		return fmt.Sprint(v)
	}
}

func normalizeScalarToString(v interface{}) string {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
