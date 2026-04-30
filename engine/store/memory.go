package store

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	activeConnections = make(map[string]*sql.DB)
	activeRedis       = make(map[string]interface{}) // Using interface{} to avoid circular dep if needed, or just *db.RedisClient
	activeCleanups    = make(map[string]func())
	connecting        = make(map[string]bool)
	memMu             sync.RWMutex
)

// AddActiveConnection saves an active *sql.DB and associates it with id.
func AddActiveConnection(id string, db *sql.DB) {
	memMu.Lock()
	defer memMu.Unlock()

	// Close existing active connection if we are reopening
	if existing, exists := activeConnections[id]; exists {
		if cleanup := activeCleanups[id]; cleanup != nil {
			cleanup()
			delete(activeCleanups, id)
		}
		existing.Close()
	}

	activeConnections[id] = db
	delete(connecting, id)
	ClearTableSizes(id)
	ClearDefaultSchema(id)
	ClearTableSizeStatus(id)
}

// SetActiveCleanup associates a cleanup function with a connection id.
// It's used for resources adjacent to *sql.DB (e.g. Teleport local proxies).
func SetActiveCleanup(id string, cleanup func()) {
	if cleanup == nil {
		return
	}
	memMu.Lock()
	defer memMu.Unlock()
	// Replace any previous cleanup.
	if prev := activeCleanups[id]; prev != nil {
		prev()
	}
	activeCleanups[id] = cleanup
}

// GetActiveConnection retrieves a connection by id. Returns nil if not found.
func GetActiveConnection(id string) *sql.DB {
	memMu.RLock()
	defer memMu.RUnlock()
	return activeConnections[id]
}

// RemoveActiveConnection removes a connection by id and closes it.
func RemoveActiveConnection(id string) error {
	memMu.Lock()
	defer memMu.Unlock()

	delete(connecting, id)
	if existing, exists := activeConnections[id]; exists {
		if cleanup := activeCleanups[id]; cleanup != nil {
			cleanup()
			delete(activeCleanups, id)
		}
		err := existing.Close()
		delete(activeConnections, id)
		ClearTableSizes(id)
		ClearDefaultSchema(id)
		ClearTableSizeStatus(id)
		if err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}
	return nil
}

// BeginConnect marks an ID as being connected to, returning false if a connect
// attempt is already in progress for the same ID.
func BeginConnect(id string) bool {
	if id == "" {
		return true
	}
	memMu.Lock()
	defer memMu.Unlock()
	if connecting[id] {
		return false
	}
	connecting[id] = true
	return true
}

// EndConnect clears the "connecting" marker for an ID.
func EndConnect(id string) {
	if id == "" {
		return
	}
	memMu.Lock()
	defer memMu.Unlock()
	delete(connecting, id)
}

// IsConnected quickly checks if a connection is tracked as active in memory.
func IsConnected(id string) bool {
	memMu.RLock()
	defer memMu.RUnlock()

	// Try pinging to be completely sure it's alive (optional, but robust)
	if db, exists := activeConnections[id]; exists {
		if err := db.Ping(); err == nil {
			return true
		}
		// If ping fails, we could potentially drop it, but we let it stay
		// as 'disconnected' below or let subsequent attempts deal with it.
	}
	return false
}

// Redis Management (Separate from SQL)

func AddRedisConnection(id string, client interface{}) {
	memMu.Lock()
	defer memMu.Unlock()
	activeRedis[id] = client
}

func GetRedisConnection(id string) interface{} {
	memMu.RLock()
	defer memMu.RUnlock()
	return activeRedis[id]
}

func RemoveRedisConnection(id string) {
	memMu.Lock()
	defer memMu.Unlock()
	delete(activeRedis, id)
}

func IsRedisConnected(id string) bool {
	memMu.Lock()
	defer memMu.Unlock()
	_, ok := activeRedis[id]
	return ok
}

// ClearRedisConnectionsForTest resets the active Redis connections map.
func ClearRedisConnectionsForTest() {
	memMu.Lock()
	defer memMu.Unlock()
	activeRedis = make(map[string]interface{})
}
