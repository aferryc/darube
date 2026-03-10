package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	foldersFile = "folders.json"
	folderMu    sync.Mutex
)

// SetFoldersFileForTest overrides foldersFile for tests. Returns a restore func.
func SetFoldersFileForTest(path string) func() {
	old := foldersFile
	foldersFile = path
	return func() { foldersFile = old }
}

// FolderConfig holds information about a connection folder.
type FolderConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReadFolders loads all folders from disk.
func ReadFolders() ([]FolderConfig, error) {
	folderMu.Lock()
	defer folderMu.Unlock()

	var folders []FolderConfig
	data, err := os.ReadFile(foldersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return folders, nil
		}
		return nil, fmt.Errorf("failed to read folders file: %w", err)
	}

	if err := json.Unmarshal(data, &folders); err != nil {
		return nil, fmt.Errorf("failed to parse folders file: %w", err)
	}

	return folders, nil
}

// WriteFolder appends or updates a single folder in folders.json.
func WriteFolder(folder FolderConfig) error {
	folderMu.Lock()
	defer folderMu.Unlock()

	var folders []FolderConfig
	data, err := os.ReadFile(foldersFile)
	if err == nil {
		json.Unmarshal(data, &folders)
	}

	updated := false
	for i, f := range folders {
		if f.ID == folder.ID {
			folders[i] = folder
			updated = true
			break
		}
	}
	if !updated {
		folders = append(folders, folder)
	}

	newData, err := json.MarshalIndent(folders, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal folders: %w", err)
	}

	return os.WriteFile(foldersFile, newData, 0644)
}

// DeleteFolder removes a folder by ID from folders.json.
// It does NOT touch connections — they will simply become uncategorized.
func DeleteFolder(id string) error {
	folderMu.Lock()
	defer folderMu.Unlock()

	data, err := os.ReadFile(foldersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var folders []FolderConfig
	if err := json.Unmarshal(data, &folders); err != nil {
		return err
	}

	var kept []FolderConfig
	for _, f := range folders {
		if f.ID != id {
			kept = append(kept, f)
		}
	}

	newData, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(foldersFile, newData, 0644)
}
