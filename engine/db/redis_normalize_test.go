package db

import "testing"

func TestNormalizeRedisValue_HGetAllPairs(t *testing.T) {
	raw := []interface{}{
		[]byte("field1"), []byte("value1"),
		"field2", int64(2),
	}
	n := normalizeRedisValue("HGETALL", raw)
	m, ok := n.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", n)
	}
	if m["field1"] != "value1" {
		t.Fatalf("field1 mismatch: %#v", m["field1"])
	}
	if m["field2"] != int64(2) {
		t.Fatalf("field2 mismatch: %#v", m["field2"])
	}
}

func TestNormalizeRedisValue_NonStringMapKeys(t *testing.T) {
	raw := map[interface{}]interface{}{
		int64(1): true,
		true:     "yes",
	}
	n := normalizeRedisValue("", raw)
	m, ok := n.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", n)
	}
	if m["1"] != true {
		t.Fatalf("1 mismatch: %#v", m["1"])
	}
	if m["true"] != "yes" {
		t.Fatalf("true mismatch: %#v", m["true"])
	}
}
