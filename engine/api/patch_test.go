package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func TestPatchConnectionFolderHandler(t *testing.T) {
	dir := t.TempDir()
	cPath := filepath.Join(dir, "connections.json")
	fPath := filepath.Join(dir, "folders.json")
	restoreC := store.SetConnectionsFileForTest(cPath)
	restoreF := store.SetFoldersFileForTest(fPath)
	defer restoreC()
	defer restoreF()

	store.WriteConnection(store.ConnectionConfig{
		ID:       "conn-1",
		DBType:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "u",
		Password: "p",
	})
	store.WriteFolder(store.FolderConfig{ID: "f1", Name: "Folder1"})

	body := bytes.NewBufferString(`{"folder_id":"f1"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/connections/conn-1/folder", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	PatchConnectionFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}

	cfg, _ := store.GetConnection("conn-1")
	if cfg.FolderID != "f1" {
		t.Errorf("folder_id not updated: %s", cfg.FolderID)
	}
}

func TestPatchConnectionFolderHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/connections//folder", nil)
	rr := httptest.NewRecorder()
	PatchConnectionFolderHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestPatchConnectionFolderHandler_InvalidBody(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	req := httptest.NewRequest(http.MethodPatch, "/api/connections/conn-1/folder", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "conn-1")
	rr := httptest.NewRecorder()
	PatchConnectionFolderHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestPatchConnectionFolderHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	restore := store.SetConnectionsFileForTest(filepath.Join(dir, "conn.json"))
	defer restore()

	body := bytes.NewBufferString(`{"folder_id":"f1"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/connections/nonexistent/folder", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	PatchConnectionFolderHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d", rr.Code)
	}
}
