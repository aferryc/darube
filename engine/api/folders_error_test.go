package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"engine/store"
)

func TestFoldersHandlers_FileErrors(t *testing.T) {
	dir := t.TempDir()
	restoreFolders := store.SetFoldersFileForTest(dir) // directory => read/write fail
	defer restoreFolders()

	req := httptest.NewRequest(http.MethodGet, "/api/folders", nil)
	rr := httptest.NewRecorder()
	ListFoldersHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("ListFoldersHandler: expected 500, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewBufferString(`{"name":"x"}`))
	rr = httptest.NewRecorder()
	CreateFolderHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("CreateFolderHandler: expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

