package store

import (
	"encoding/json"
	"os"
)

var workspaceFile = "workspace.json"

// SetWorkspaceFileForTest overrides workspaceFile for tests. Returns a restore func.
func SetWorkspaceFileForTest(path string) func() {
	old := workspaceFile
	workspaceFile = path
	return func() { workspaceFile = old }
}

// TabState represents the minimal state of a query tab needed for persistence
type TabState struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Type         string `json:"type,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
}

// WorkspaceState represents the list of saved tabs
type WorkspaceState struct {
	Tabs []TabState `json:"tabs"`
}

// LoadWorkspace loads the saved workspace tabs from workspace.json
func LoadWorkspace() (WorkspaceState, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	var ws WorkspaceState
	ws.Tabs = []TabState{} // ensure it's an empty array, not null

	data, err := os.ReadFile(workspaceFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ws, nil
		}
		return ws, err
	}

	if err := json.Unmarshal(data, &ws); err != nil {
		return ws, err
	}

	return ws, nil
}

// SaveWorkspace writes the given tabs array to workspace.json
func SaveWorkspace(ws WorkspaceState) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(workspaceFile, data, 0644)
}
