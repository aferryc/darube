package metadata

import (
	"database/sql"
	"fmt"
)

type DatabaseInfo struct {
	Name string `json:"name"`
}

type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Primary bool     `json:"primary"`
}

type EntityInfo struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"` // "table" or "view"
	Columns []ColumnInfo `json:"columns,omitempty"`
	Indexes []string     `json:"indexes,omitempty"`
}

type SchemaInfo struct {
	Name   string       `json:"name"`
	Tables []EntityInfo `json:"tables"`
}

// GetDatabases executes a driver-specific query to list available databases.
func GetDatabases(dbType string, db *sql.DB) ([]DatabaseInfo, error) {
	var query string
	switch dbType {
	case "postgres":
		query = "SELECT datname FROM pg_database WHERE datistemplate = false"
	case "mysql":
		query = "SHOW DATABASES"
	case "sqlserver":
		query = "SELECT name FROM sys.databases"
	case "oracle":
		query = "SELECT SYS_CONTEXT('USERENV','DB_NAME') FROM dual"
	case "sqlite":
		// SQLite is a single-file database. Expose a conventional name so the UI has something to show.
		return []DatabaseInfo{{Name: "main"}}, nil
	default:
		return nil, fmt.Errorf("unsupported database type for metadata fetching: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query to fetch databases failed: %w", err)
	}
	defer rows.Close()

	var dbs []DatabaseInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		dbs = append(dbs, DatabaseInfo{Name: name})
	}
	return dbs, nil
}
