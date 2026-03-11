package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"engine/store"
)

func TestListConnectionsHandler_ReadError(t *testing.T) {
	dir := t.TempDir()
	// Point connections "file" at a directory so ReadFile fails.
	restore := store.SetConnectionsFileForTest(dir)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/connections", nil)
	rr := httptest.NewRecorder()
	ListConnectionsHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

