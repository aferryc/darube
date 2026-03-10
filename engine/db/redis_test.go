package db

import (
	"encoding/json"
	"fmt"
	"testing"
)

// NOTE: This requires a running Redis or a mock.
// For TDD without a real Redis, we can use redismock,
// but let's see if we can at least verify the structure.

func TestRedisResultFormatting(t *testing.T) {
	// Test JSON detection
	client := &RedisClient{}
	res, err := client.handleFormatting(`{"status": "ok", "code": 200}`)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res.DataType != "json" {
		t.Errorf("Expected json data type, got %s", res.DataType)
	}

	// Test string detection
	res, _ = client.handleFormatting("hello world")
	if res.DataType != "string" {
		t.Errorf("Expected string data type, got %s", res.DataType)
	}

	// Test nil detection (missing key etc.)
	res, _ = client.handleFormatting(nil)
	if res.DataType != "nil" {
		t.Errorf("Expected nil data type, got %s", res.DataType)
	}
}

// Internal helper for testing without live connection
func (c *RedisClient) handleFormatting(val interface{}) (*RedisResult, error) {
	result := &RedisResult{Value: val}
	switch v := val.(type) {
	case nil:
		result.DataType = "nil"
	case string:
		result.DataType = "string"
		var js interface{}
		if err := json.Unmarshal([]byte(v), &js); err == nil {
			result.DataType = "json"
			result.Value = js
		}
	case int64:
		result.DataType = "integer"
	default:
		result.DataType = fmt.Sprintf("%T", v)
	}
	return result, nil
}
