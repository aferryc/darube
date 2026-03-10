package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkspace_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	restore := SetWorkspaceFileForTest(path)
	defer restore()

	ws, err := LoadWorkspace()
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if ws.Tabs == nil {
		t.Error("expected Tabs to be non-nil empty slice")
	}
	if len(ws.Tabs) != 0 {
		t.Errorf("expected empty tabs, got %d", len(ws.Tabs))
	}
}

func TestLoadWorkspace_NotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	restore := SetWorkspaceFileForTest(path)
	defer restore()

	ws, err := LoadWorkspace()
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if len(ws.Tabs) != 0 {
		t.Errorf("expected empty tabs, got %d", len(ws.Tabs))
	}
}

func TestLoadWorkspace_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	restore := SetWorkspaceFileForTest(path)
	defer restore()

	data := `{"tabs":[{"id":"tab-1","name":"Query 1","query":"SELECT 1","type":"query","connection_id":"c1"},{"id":"tab-2","name":"Query 2","query":"SELECT 2","type":"query","connection_id":"c2"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := LoadWorkspace()
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if len(ws.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(ws.Tabs))
	}
	if ws.Tabs[0].ID != "tab-1" || ws.Tabs[0].Query != "SELECT 1" {
		t.Errorf("unexpected tab: %+v", ws.Tabs[0])
	}
	if ws.Tabs[0].Type != "query" || ws.Tabs[0].ConnectionID != "c1" {
		t.Errorf("unexpected tab metadata: %+v", ws.Tabs[0])
	}
}

func TestLoadWorkspace_Corrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	restore := SetWorkspaceFileForTest(path)
	defer restore()

	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadWorkspace()
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestSaveWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.json")
	restore := SetWorkspaceFileForTest(path)
	defer restore()

	ws := WorkspaceState{
		Tabs: []TabState{
			{ID: "tab-1", Name: "Query 1", Query: "SELECT * FROM users", Type: "query", ConnectionID: "c1"},
			{ID: "tab-2", Name: "Query 2", Query: "SELECT * FROM orders", Type: "query", ConnectionID: "c2"},
		},
	}
	if err := SaveWorkspace(ws); err != nil {
		t.Fatalf("SaveWorkspace: %v", err)
	}

	loaded, err := LoadWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(loaded.Tabs))
	}
	if loaded.Tabs[0].Query != "SELECT * FROM users" {
		t.Errorf("unexpected query: %s", loaded.Tabs[0].Query)
	}
}
