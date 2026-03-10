package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	connectionsFile      = "connections.json"
	redisConnectionsFile = "redis_connections.json"
	fileMu               sync.Mutex
)

// SetConnectionsFileForTest overrides connectionsFile for tests. Returns a restore func.
func SetConnectionsFileForTest(path string) func() {
	old := connectionsFile
	connectionsFile = path
	return func() { connectionsFile = old }
}

// ReadConnections loads connections from the JSON file.
func ReadConnections() ([]ConnectionConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []ConnectionConfig
	data, err := os.ReadFile(connectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return connections, nil
		}
		return nil, fmt.Errorf("failed to read connections file: %w", err)
	}

	if err := json.Unmarshal(data, &connections); err != nil {
		return nil, fmt.Errorf("failed to parse connections file: %w", err)
	}

	return connections, nil
}

// GetConnection returns a single connection config by ID.
func GetConnection(id string) (*ConnectionConfig, error) {
	conns, err := ReadConnections()
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("connection '%s' not found", id)
}

// WriteConnection append or updates a single connection config to the JSON file.
func WriteConnection(config ConnectionConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []ConnectionConfig
	
	// Ensure the parent directory exists if connectionsFile is a path
	if dir := filepath.Dir(connectionsFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	data, err := os.ReadFile(connectionsFile)
	if err == nil {
		// Ignore unmarshal error on corrupt file, just overwrite
		json.Unmarshal(data, &connections)
	}

	// Update if exists, else append
	updated := false
	for i, c := range connections {
		if c.ID == config.ID {
			connections[i] = config
			updated = true
			break
		}
	}

	if !updated {
		connections = append(connections, config)
	}

	newData, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal connections: %w", err)
	}

	if err := os.WriteFile(connectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write connections file: %w", err)
	}

	return nil
}

// DeleteConnection removes a connection by ID from the JSON file.
func DeleteConnection(id string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []ConnectionConfig
	data, err := os.ReadFile(connectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read connections file: %w", err)
	}

	if err := json.Unmarshal(data, &connections); err != nil {
		return fmt.Errorf("failed to parse connections file: %w", err)
	}

	var newConnections []ConnectionConfig
	for _, c := range connections {
		if c.ID != id {
			newConnections = append(newConnections, c)
		}
	}

	newData, err := json.MarshalIndent(newConnections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal connections: %w", err)
	}

	if err := os.WriteFile(connectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write connections file: %w", err)
	}

	return nil
}
// ReadRedisConnections loads Redis connections from the JSON file.
func ReadRedisConnections() ([]RedisConfig, error) {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []RedisConfig
	data, err := os.ReadFile(redisConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return connections, nil
		}
		return nil, fmt.Errorf("failed to read redis connections file: %w", err)
	}

	if err := json.Unmarshal(data, &connections); err != nil {
		return nil, fmt.Errorf("failed to parse redis connections file: %w", err)
	}

	return connections, nil
}

// GetRedisConfig returns a single Redis connection config by ID.
func GetRedisConfig(id string) (*RedisConfig, error) {
	conns, err := ReadRedisConnections()
	if err != nil {
		return nil, err
	}
	for _, c := range conns {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("redis connection '%s' not found", id)
}

// WriteRedisConnection append or updates a single Redis connection config to the JSON file.
func WriteRedisConnection(config RedisConfig) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []RedisConfig
	
	if dir := filepath.Dir(redisConnectionsFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	data, err := os.ReadFile(redisConnectionsFile)
	if err == nil {
		json.Unmarshal(data, &connections)
	}

	updated := false
	for i, c := range connections {
		if c.ID == config.ID {
			connections[i] = config
			updated = true
			break
		}
	}

	if !updated {
		connections = append(connections, config)
	}

	newData, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal redis connections: %w", err)
	}

	if err := os.WriteFile(redisConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write redis connections file: %w", err)
	}

	return nil
}

// DeleteRedisConnection removes a Redis connection by ID from the JSON file.
func DeleteRedisConnection(id string) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	var connections []RedisConfig
	data, err := os.ReadFile(redisConnectionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read redis connections file: %w", err)
	}

	if err := json.Unmarshal(data, &connections); err != nil {
		return fmt.Errorf("failed to parse redis connections file: %w", err)
	}

	var newConnections []RedisConfig
	for _, c := range connections {
		if c.ID != id {
			newConnections = append(newConnections, c)
		}
	}

	newData, err := json.MarshalIndent(newConnections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal redis connections: %w", err)
	}

	if err := os.WriteFile(redisConnectionsFile, newData, 0644); err != nil {
		return fmt.Errorf("failed to write redis connections file: %w", err)
	}

	return nil
}
