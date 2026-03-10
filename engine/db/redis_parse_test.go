package db

import (
	"reflect"
	"testing"
)

func TestParseRedisCommandArgs_Basic(t *testing.T) {
	got, err := parseRedisCommandArgs("HGETALL v2:cashupdate:0909811")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"HGETALL", "v2:cashupdate:0909811"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestParseRedisCommandArgs_QuotesAndEscapes(t *testing.T) {
	got, err := parseRedisCommandArgs("SET k \"hello world\\n\"")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"SET", "k", "hello world\n"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

