package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFolders_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	folders, err := ReadFolders()
	if err != nil {
		t.Fatalf("ReadFolders: %v", err)
	}
	if len(folders) != 0 {
		t.Errorf("expected empty, got %d folders", len(folders))
	}
}

func TestReadFolders_NotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	folders, err := ReadFolders()
	if err != nil {
		t.Fatalf("ReadFolders: %v", err)
	}
	if len(folders) != 0 {
		t.Errorf("expected empty, got %d folders", len(folders))
	}
}

func TestReadFolders_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	data := `[{"id":"f1","name":"Production"},{"id":"f2","name":"Dev"}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	folders, err := ReadFolders()
	if err != nil {
		t.Fatalf("ReadFolders: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(folders))
	}
	if folders[0].ID != "f1" || folders[0].Name != "Production" {
		t.Errorf("unexpected folder: %+v", folders[0])
	}
}

func TestReadFolders_Corrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadFolders()
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestWriteFolder_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	folder := FolderConfig{ID: "f1", Name: "New Folder"}
	if err := WriteFolder(folder); err != nil {
		t.Fatalf("WriteFolder: %v", err)
	}

	folders, err := ReadFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Name != "New Folder" {
		t.Errorf("expected 1 folder, got %+v", folders)
	}
}

func TestWriteFolder_Update(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	data := `[{"id":"f1","name":"Old Name"}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFolder(FolderConfig{ID: "f1", Name: "Updated Name"}); err != nil {
		t.Fatalf("WriteFolder: %v", err)
	}

	folders, err := ReadFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Name != "Updated Name" {
		t.Errorf("expected updated name, got %+v", folders)
	}
}

func TestDeleteFolder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	data := `[{"id":"f1","name":"First"},{"id":"f2","name":"Second"}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteFolder("f1"); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}

	folders, err := ReadFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].ID != "f2" {
		t.Errorf("expected f2 only, got %+v", folders)
	}
}

func TestDeleteFolder_NotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "folders.json")
	restore := SetFoldersFileForTest(path)
	defer restore()

	if err := DeleteFolder("nonexistent"); err != nil {
		t.Errorf("DeleteFolder on empty file should succeed: %v", err)
	}
}
