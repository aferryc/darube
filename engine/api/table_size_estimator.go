package api

import (
	"database/sql"
	"strings"

	"engine/store"
)

func startTableSizeEstimator(connID string, config store.ConnectionConfig) {
	store.SetTableSizeStatus(connID, "running", "")
	go func() {
		runTableSizeEstimator(connID, config)
	}()
}

func runTableSizeEstimator(connID string, config store.ConnectionConfig) {
	db := store.GetActiveConnection(connID)
	if db == nil {
		return
	}

	store.SetTableSizeStatus(connID, "running", "")

	switch config.DBType {
	case "postgres", "postgresql":
		if err := estimatePostgresTableSizes(connID, db); err != nil {
			store.SetTableSizeStatus(connID, "error", err.Error())
			return
		}
		store.SetTableSizeStatus(connID, "ready", "")
	case "mysql", "mariadb":
		if err := estimateMySQLTableSizes(connID, db); err != nil {
			store.SetTableSizeStatus(connID, "error", err.Error())
			return
		}
		store.SetTableSizeStatus(connID, "ready", "")
	default:
		store.SetTableSizeStatus(connID, "unsupported", "")
		return
	}
}

func estimatePostgresTableSizes(connID string, db *sql.DB) error {
	const query = `
		SELECT n.nspname AS schema_name,
		       c.relname AS table_name,
		       c.relpages * current_setting('block_size')::int8 AS size_bytes
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p')
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	sizes := make([]store.TableSize, 0, 128)
	for rows.Next() {
		var schema, table string
		var size sql.NullInt64
		if err := rows.Scan(&schema, &table, &size); err != nil {
			return err
		}
		if size.Valid {
			sizes = append(sizes, store.TableSize{Schema: schema, Table: table, SizeBytes: size.Int64})
		}
	}
	store.SetTableSizes(connID, sizes)

	var defaultSchema sql.NullString
	if err := db.QueryRow("SELECT current_schema()").Scan(&defaultSchema); err == nil {
		if defaultSchema.Valid && strings.TrimSpace(defaultSchema.String) != "" {
			store.SetDefaultSchema(connID, defaultSchema.String)
		}
	}
	return nil
}

func estimateMySQLTableSizes(connID string, db *sql.DB) error {
	const query = `
		SELECT table_schema,
		       table_name,
		       data_length + index_length AS size_bytes
		FROM information_schema.tables
		WHERE table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	sizes := make([]store.TableSize, 0, 256)
	for rows.Next() {
		var schema, table string
		var size sql.NullInt64
		if err := rows.Scan(&schema, &table, &size); err != nil {
			return err
		}
		if size.Valid {
			sizes = append(sizes, store.TableSize{Schema: schema, Table: table, SizeBytes: size.Int64})
		}
	}
	if len(sizes) > 0 {
		store.SetTableSizes(connID, sizes)
	}
	return nil
}
