package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"
)

func setupFoldersTest(t *testing.T) func() {
	dir := t.TempDir()
	fPath := filepath.Join(dir, "folders.json")
	cPath := filepath.Join(dir, "connections.json")
	rPath := filepath.Join(dir, "redis_connections.json")
	restoreF := store.SetFoldersFileForTest(fPath)
	restoreC := store.SetConnectionsFileForTest(cPath)
	restoreR := store.SetRedisConnectionsFileForTest(rPath)
	return func() {
		restoreF()
		restoreC()
		restoreR()
	}
}

func TestListFoldersHandler(t *testing.T) {
	defer setupFoldersTest(t)()

	req := httptest.NewRequest(http.MethodGet, "/api/folders", nil)
	rr := httptest.NewRecorder()
	ListFoldersHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("success: %v", resp["success"])
	}
	folders, _ := resp["folders"].([]interface{})
	if len(folders) != 0 {
		t.Errorf("expected empty folders, got %d", len(folders))
	}
}

func TestCreateFolderHandler(t *testing.T) {
	defer setupFoldersTest(t)()

	body := bytes.NewBufferString(`{"name":"Production"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/folders", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	CreateFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("success: %v", resp["success"])
	}
	folder, ok := resp["folder"].(map[string]interface{})
	if !ok {
		t.Fatalf("folder not found in response")
	}
	if folder["name"] != "Production" {
		t.Errorf("folder name: %v", folder["name"])
	}
	if folder["id"] == nil || folder["id"] == "" {
		t.Error("folder id should be set")
	}
}

func TestCreateFolderHandler_MissingName(t *testing.T) {
	defer setupFoldersTest(t)()

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/folders", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	CreateFolderHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestUpdateFolderHandler(t *testing.T) {
	defer setupFoldersTest(t)()

	// Create folder first
	store.WriteFolder(store.FolderConfig{ID: "f1", Name: "Old"})

	body := bytes.NewBufferString(`{"name":"Updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/folders/f1", body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "f1")
	rr := httptest.NewRecorder()
	UpdateFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	folders, _ := store.ReadFolders()
	if len(folders) != 1 || folders[0].Name != "Updated" {
		t.Errorf("folder not updated: %+v", folders)
	}
}

func TestUpdateFolderHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/folders/", nil)
	rr := httptest.NewRecorder()
	UpdateFolderHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestDeleteFolderHandler(t *testing.T) {
	defer setupFoldersTest(t)()

	store.WriteFolder(store.FolderConfig{ID: "f1", Name: "ToDelete"})

	req := httptest.NewRequest(http.MethodDelete, "/api/folders/f1", nil)
	req.SetPathValue("id", "f1")
	rr := httptest.NewRecorder()
	DeleteFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	folders, _ := store.ReadFolders()
	if len(folders) != 0 {
		t.Errorf("folder not deleted: %+v", folders)
	}
}

func TestDeleteFolderHandler_MissingId(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/folders/", nil)
	rr := httptest.NewRecorder()
	DeleteFolderHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestDeleteFolderHandler_ClearsFolderLinks(t *testing.T) {
	defer setupFoldersTest(t)()

	folderID := "f1"
	store.WriteFolder(store.FolderConfig{ID: folderID, Name: "ToDelete"})
	_ = store.WriteConnection(store.ConnectionConfig{ID: "c1", ConnectionName: "db", DBType: "sqlite", FilePath: ":memory:", FolderID: folderID})
	_ = store.WriteRedisConnection(store.RedisConfig{ID: "r1", ConnectionName: "cache", Host: "localhost", Port: 6379, FolderID: folderID})

	req := httptest.NewRequest(http.MethodDelete, "/api/folders/"+folderID, nil)
	req.SetPathValue("id", folderID)
	rr := httptest.NewRecorder()
	DeleteFolderHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	c, err := store.GetConnection("c1")
	if err != nil || c == nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if c.FolderID != "" {
		t.Fatalf("expected connection folder_id cleared, got %q", c.FolderID)
	}
	rc, err := store.GetRedisConfig("r1")
	if err != nil || rc == nil {
		t.Fatalf("GetRedisConfig: %v", err)
	}
	if rc.FolderID != "" {
		t.Fatalf("expected redis folder_id cleared, got %q", rc.FolderID)
	}
}
