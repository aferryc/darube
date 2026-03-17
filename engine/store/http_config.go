package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var httpConnectionsFile = "http_connections.json"

// SetHTTPConnectionsFileForTest overrides httpConnectionsFile for tests. Returns a restore func.
func SetHTTPConnectionsFileForTest(path string) func() {
	old := httpConnectionsFile
	httpConnectionsFile = path
	return func() { httpConnectionsFile = old }
}

type HTTPHeader struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type HTTPAuth struct {
	Type        string `json:"type"` // "none" | "bearer" | "basic"
	BearerToken string `json:"bearer_token,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

type HTTPConfig struct {
	ID             string       `json:"id"`
	ConnectionName string       `json:"connection_name"`
	BaseURL        string       `json:"base_url"`
	DefaultHeaders []HTTPHeader `json:"default_headers,omitempty"`
	Auth           HTTPAuth     `json:"auth,omitempty"`
	FolderID       string       `json:"folder_id,omitempty"`
}

func ReadHTTPConnections() ([]HTTPConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []HTTPConfig
	data, err := os.ReadFile(httpConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return conns, nil
		}
		return nil, fmt.Errorf("failed to read http connections file: %w", err)
	}
	if err := json.Unmarshal(data, &conns); err != nil {
		return nil, fmt.Errorf("failed to parse http connections file: %w", err)
	}
	return conns, nil
}

func GetHTTPConfig(id string) (*HTTPConfig, error) {
	conns, err := ReadHTTPConnections()
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("http connection '%s' not found", id)
}

func WriteHTTPConnection(cfg HTTPConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []HTTPConfig
	if dir := filepath.Dir(httpConnectionsFile); dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := os.ReadFile(httpConnectionsFile)
	if err == nil {
		_ = json.Unmarshal(data, &conns)
	}

	updated := false
	for i := range conns {
		if conns[i].ID == cfg.ID {
			conns[i] = cfg
			updated = true
			break
		}
	}
	if !updated {
		conns = append(conns, cfg)
	}

	newData, err := json.MarshalIndent(conns, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal http connections: %w", err)
	}
	if err := os.WriteFile(httpConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write http connections file: %w", err)
	}
	return nil
}

func DeleteHTTPConnection(id string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []HTTPConfig
	data, err := os.ReadFile(httpConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read http connections file: %w", err)
	}
	if err := json.Unmarshal(data, &conns); err != nil {
		return fmt.Errorf("failed to parse http connections file: %w", err)
	}

	out := make([]HTTPConfig, 0, len(conns))
	for _, c := range conns {
		if c.ID != id {
			out = append(out, c)
		}
	}
	newData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal http connections: %w", err)
	}
	if err := os.WriteFile(httpConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write http connections file: %w", err)
	}
	return nil
}

