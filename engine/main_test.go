package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnableCORS_PreflightAndPassThrough(t *testing.T) {
	nextCalled := false
	h := enableCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	}))

	// Preflight request should short-circuit.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/connections", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if nextCalled {
		t.Fatalf("expected next not called on OPTIONS")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected CORS headers set")
	}

	// Non-OPTIONS should pass through.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected passthrough status, got %d", rr.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next to be called")
	}
}

func TestRun_CallsListenAndServe(t *testing.T) {
	t.Setenv("PORT", "12345")

	prev := listenAndServe
	t.Cleanup(func() { listenAndServe = prev })

	called := false
	listenAndServe = func(addr string, handler http.Handler) error {
		called = true
		if addr != ":12345" {
			t.Fatalf("unexpected addr: %s", addr)
		}

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/api/connections", nil)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 from preflight, got %d", rr.Code)
		}
		return http.ErrServerClosed
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatalf("expected listenAndServe to be called")
	}
}

