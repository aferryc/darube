package store

import (
	"strings"
	"sync"
	"time"
)

type TableSize struct {
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	SizeBytes int64  `json:"size_bytes"`
}

type TableSizeStatus struct {
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

var (
	tableSizesByConn    = make(map[string]map[string]TableSize)
	tableSizesUpdated   = make(map[string]time.Time)
	tableSizesStatus    = make(map[string]TableSizeStatus)
	defaultSchemaByConn = make(map[string]string)
	tableSizeMu         sync.RWMutex
)

func normalizeKey(schema, table string) string {
	s := strings.ToLower(strings.TrimSpace(schema))
	t := strings.ToLower(strings.TrimSpace(table))
	if s == "" {
		return t
	}
	return s + "." + t
}

func SetTableSizes(connID string, sizes []TableSize) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	next := make(map[string]TableSize, len(sizes))
	for _, size := range sizes {
		key := normalizeKey(size.Schema, size.Table)
		if key == "" {
			continue
		}
		next[key] = size
	}
	tableSizesByConn[connID] = next
	tableSizesUpdated[connID] = time.Now()
}

func GetTableSize(connID, schema, table string) (TableSize, bool) {
	tableSizeMu.RLock()
	defer tableSizeMu.RUnlock()
	tables := tableSizesByConn[connID]
	if tables == nil {
		return TableSize{}, false
	}
	size, ok := tables[normalizeKey(schema, table)]
	return size, ok
}

func GetTableSizes(connID string) ([]TableSize, time.Time) {
	tableSizeMu.RLock()
	defer tableSizeMu.RUnlock()
	tables := tableSizesByConn[connID]
	sizes := make([]TableSize, 0, len(tables))
	for _, size := range tables {
		sizes = append(sizes, size)
	}
	return sizes, tableSizesUpdated[connID]
}

func GetTableSizeCount(connID string) int {
	tableSizeMu.RLock()
	defer tableSizeMu.RUnlock()
	return len(tableSizesByConn[connID])
}

func SetTableSizeStatus(connID, status, errMsg string) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	current := tableSizesStatus[connID]
	current.Status = status
	current.Error = errMsg
	if updated := tableSizesUpdated[connID]; !updated.IsZero() {
		current.UpdatedAt = updated
	}
	tableSizesStatus[connID] = current
}

func GetTableSizeStatus(connID string) TableSizeStatus {
	tableSizeMu.RLock()
	defer tableSizeMu.RUnlock()
	return tableSizesStatus[connID]
}

func ClearTableSizes(connID string) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	delete(tableSizesByConn, connID)
	delete(tableSizesUpdated, connID)
}

func SetDefaultSchema(connID, schema string) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	if strings.TrimSpace(schema) == "" {
		return
	}
	defaultSchemaByConn[connID] = strings.TrimSpace(schema)
}

func GetDefaultSchema(connID string) string {
	tableSizeMu.RLock()
	defer tableSizeMu.RUnlock()
	return defaultSchemaByConn[connID]
}

func ClearDefaultSchema(connID string) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	delete(defaultSchemaByConn, connID)
}

func ClearTableSizeStatus(connID string) {
	tableSizeMu.Lock()
	defer tableSizeMu.Unlock()
	delete(tableSizesStatus, connID)
}
