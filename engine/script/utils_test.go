package script

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type errProvider struct{}

func (errProvider) Conn(context.Context, string) (ConnHandle, error) {
	return nil, context.Canceled
}

func TestScriptUtils_Basics(t *testing.T) {
	eng := New(errProvider{})
	out, err := eng.Run(context.Background(), `({ u: utils.uuidv7(), t: utils.now(), ms: utils.nowUnixMs() })`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object, got %T (%#v)", out, out)
	}
	u, _ := m["u"].(string)
	if u == "" {
		t.Fatalf("expected uuidv7 string, got %#v", m["u"])
	}
	parsed, err := uuid.Parse(u)
	if err != nil {
		t.Fatalf("uuid parse: %v (%q)", err, u)
	}
	if parsed.Version() != 7 {
		t.Fatalf("expected uuidv7, got v%d (%q)", parsed.Version(), u)
	}
	if _, ok := m["t"].(string); !ok {
		t.Fatalf("expected now() string, got %T (%#v)", m["t"], m["t"])
	}
	if _, ok := m["ms"].(int64); !ok {
		t.Fatalf("expected nowUnixMs() int64, got %T (%#v)", m["ms"], m["ms"])
	}
}

func TestScriptUtils_SleepCanceled(t *testing.T) {
	eng := New(errProvider{})
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	start := time.Now()
	_, err := eng.Run(ctx, `sleep(10000); 1`)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error")
	}
	if elapsed > 600*time.Millisecond {
		t.Fatalf("sleep did not cancel fast enough: %s", elapsed)
	}
}
