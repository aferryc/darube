package store

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	activeConnections = make(map[string]*sql.DB)
	activeRedis       = make(map[string]interface{}) // Using interface{} to avoid circular dep if needed, or just *db.RedisClient
	memMu             sync.RWMutex
)

// AddActiveConnection saves an active *sql.DB and associates it with id.
func AddActiveConnection(id string, db *sql.DB) {
	memMu.Lock()
	defer memMu.Unlock()
	
	// Close existing active connection if we are reopening
	if existing, exists := activeConnections[id]; exists {
		existing.Close()
	}

	activeConnections[id] = db
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

	if existing, exists := activeConnections[id]; exists {
		err := existing.Close()
		delete(activeConnections, id)
		if err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}
	return nil
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
