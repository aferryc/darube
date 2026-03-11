package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendJSONResponse_MarshalError(t *testing.T) {
	rr := httptest.NewRecorder()
	sendJSONResponse(rr, map[string]interface{}{"bad": make(chan int)}, http.StatusOK)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "Failed to marshal response") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

