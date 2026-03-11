package api

import "testing"

func TestRedisCellString_Variants(t *testing.T) {
	if redisCellString(nil) != "" {
		t.Fatalf("nil should be empty")
	}
	if redisCellString("x") != "x" {
		t.Fatalf("string passthrough")
	}
	if redisCellString(map[string]interface{}{"a": 1}) == "" {
		t.Fatalf("struct/map should stringify")
	}
	// json.Marshal fails for channels -> fallback fmt.Sprint path.
	if redisCellString(make(chan int)) == "" {
		t.Fatalf("marshal error fallback should not be empty")
	}
}

