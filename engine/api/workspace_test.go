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

func setupWorkspaceTest(t *testing.T) func() {
	dir := t.TempDir()
	wPath := filepath.Join(dir, "workspace.json")
	restore := store.SetWorkspaceFileForTest(wPath)
	return restore
}

func TestGetWorkspaceHandler(t *testing.T) {
	defer setupWorkspaceTest(t)()

	req := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	rr := httptest.NewRecorder()
	GetWorkspaceHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d", rr.Code)
	}
	var ws store.WorkspaceState
	if err := json.NewDecoder(rr.Body).Decode(&ws); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ws.Tabs == nil {
		t.Error("Tabs should not be nil")
	}
}

func TestSaveWorkspaceHandler(t *testing.T) {
	defer setupWorkspaceTest(t)()

	ws := store.WorkspaceState{
		Tabs: []store.TabState{
			{ID: "tab-1", Name: "Query 1", Query: "SELECT 1"},
		},
	}
	body, _ := json.Marshal(ws)
	req := httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	SaveWorkspaceHandler(rr, req)

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

	// Verify persisted
	loaded, err := store.LoadWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Tabs) != 1 || loaded.Tabs[0].Query != "SELECT 1" {
		t.Errorf("workspace not saved: %+v", loaded)
	}
}

func TestSaveWorkspaceHandler_InvalidPayload(t *testing.T) {
	defer setupWorkspaceTest(t)()

	req := httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	SaveWorkspaceHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rr.Code)
	}
}

func TestGetWorkspaceHandler_LoadError(t *testing.T) {
	dir := t.TempDir()
	// Point workspace "file" at a directory so ReadFile fails.
	restore := store.SetWorkspaceFileForTest(dir)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/workspace", nil)
	rr := httptest.NewRecorder()
	GetWorkspaceHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSaveWorkspaceHandler_SaveError(t *testing.T) {
	dir := t.TempDir()
	// Point workspace "file" at a directory so WriteFile fails.
	restore := store.SetWorkspaceFileForTest(dir)
	defer restore()

	ws := store.WorkspaceState{Tabs: []store.TabState{{ID: "t1", Name: "n", Query: "q"}}}
	body, _ := json.Marshal(ws)
	req := httptest.NewRequest(http.MethodPost, "/api/workspace", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	SaveWorkspaceHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}
