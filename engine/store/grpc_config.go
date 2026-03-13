package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var grpcConnectionsFile = "grpc_connections.json"

// SetGRPCConnectionsFileForTest overrides grpcConnectionsFile for tests. Returns a restore func.
func SetGRPCConnectionsFileForTest(path string) func() {
	old := grpcConnectionsFile
	grpcConnectionsFile = path
	return func() { grpcConnectionsFile = old }
}

type GRPCAuth struct {
	Type        string `json:"type"` // "none" | "bearer" | "basic"
	BearerToken string `json:"bearer_token,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

type GRPCConfig struct {
	ID             string   `json:"id"`
	ConnectionName string   `json:"connection_name"`
	Address        string   `json:"address"` // host:port
	TLS            bool     `json:"tls"`
	InsecureTLS    bool     `json:"insecure_tls"` // skip verify
	ServerName     string   `json:"server_name,omitempty"`
	Auth           GRPCAuth `json:"auth,omitempty"`
	FolderID       string   `json:"folder_id,omitempty"`
}

func ReadGRPCConnections() ([]GRPCConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []GRPCConfig
	data, err := os.ReadFile(grpcConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return conns, nil
		}
		return nil, fmt.Errorf("failed to read grpc connections file: %w", err)
	}
	if err := json.Unmarshal(data, &conns); err != nil {
		return nil, fmt.Errorf("failed to parse grpc connections file: %w", err)
	}
	return conns, nil
}

func GetGRPCConfig(id string) (*GRPCConfig, error) {
	conns, err := ReadGRPCConnections()
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("grpc connection '%s' not found", id)
}

func WriteGRPCConnection(cfg GRPCConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []GRPCConfig
	if dir := filepath.Dir(grpcConnectionsFile); dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := os.ReadFile(grpcConnectionsFile)
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
		return fmt.Errorf("failed to marshal grpc connections: %w", err)
	}
	if err := os.WriteFile(grpcConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write grpc connections file: %w", err)
	}
	return nil
}

func DeleteGRPCConnection(id string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var conns []GRPCConfig
	data, err := os.ReadFile(grpcConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read grpc connections file: %w", err)
	}
	if err := json.Unmarshal(data, &conns); err != nil {
		return fmt.Errorf("failed to parse grpc connections file: %w", err)
	}

	out := make([]GRPCConfig, 0, len(conns))
	for _, c := range conns {
		if c.ID != id {
			out = append(out, c)
		}
	}
	newData, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal grpc connections: %w", err)
	}
	if err := os.WriteFile(grpcConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write grpc connections file: %w", err)
	}
	return nil
}

