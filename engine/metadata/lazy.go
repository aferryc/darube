package metadata

import (
	"database/sql"
	"fmt"
	"strings"
)

func escapePGString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// GetSchemas returns only the schema/database names that exist in the database.
func GetSchemas(dbType string, db *sql.DB) ([]SchemaInfo, error) {
	var results []SchemaInfo
	var query string

	switch dbType {
	case "postgres":
		query = `SELECT schema_name
				 FROM information_schema.schemata
				 WHERE schema_name NOT IN ('information_schema', 'pg_catalog')
				 ORDER BY schema_name;`
	case "mysql":
		// MySQL schemas are essentially databases, but since we connect via user DSN,
		// typical behavior is tracking the connected DATABASE() as the "schema".
		query = `SELECT DATABASE() as schema_name;`
	case "sqlserver":
		query = `SELECT name AS schema_name 
				 FROM sys.schemas 
				 WHERE name NOT IN ('sys', 'INFORMATION_SCHEMA')
				 ORDER BY name;`
	default:
		return nil, fmt.Errorf("unsupported database type for schema fetching: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName sql.NullString
		if err := rows.Scan(&schemaName); err != nil {
			continue
		}
		if schemaName.Valid && schemaName.String != "" {
			results = append(results, SchemaInfo{Name: schemaName.String, Tables: []EntityInfo{}})
		}
	}

	return results, nil
}

// GetTablesList returns all tables and views for a given schema.
func GetTablesList(dbType string, db *sql.DB, schemaName string) ([]EntityInfo, error) {
	var results []EntityInfo
	var query string
	var args []interface{}

	switch dbType {
	case "postgres":
		query = fmt.Sprintf(`SELECT table_name, table_type
				 FROM information_schema.tables
				 WHERE table_schema = '%s'
				 ORDER BY table_name;`, escapePGString(schemaName))
	case "mysql":
		query = `SELECT TABLE_NAME, TABLE_TYPE
				 FROM information_schema.tables
				 WHERE TABLE_SCHEMA = ?
				 ORDER BY TABLE_NAME;`
		args = append(args, schemaName)
	case "sqlserver":
		query = `SELECT t.name AS table_name, t.type_desc AS table_type
				 FROM sys.objects t
				 JOIN sys.schemas s ON t.schema_id = s.schema_id
				 WHERE s.name = @p1 AND t.type IN ('U', 'V')
				 ORDER BY t.name;`
		args = append(args, sql.Named("p1", schemaName))
	default:
		return nil, fmt.Errorf("unsupported database type for tables fetching: %s", dbType)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tName, pType sql.NullString
		if err := rows.Scan(&tName, &pType); err != nil {
			continue
		}
		
		if tName.Valid {
			objType := "table"
			if pType.Valid && (pType.String == "VIEW" || pType.String == "SQL_INLINE_TABLE_VALUED_FUNCTION" || pType.String == "SQL_TABLE_VALUED_FUNCTION") {
				objType = "view"
			}
			results = append(results, EntityInfo{
				Name:    tName.String,
				Type:    objType,
				Columns: []ColumnInfo{},
				Indexes: []string{},
			})
		}
	}

	// Fetch indexes for Postgres (you can expand for others easily)
	if dbType == "postgres" && len(results) > 0 {
		idxQuery := fmt.Sprintf(`SELECT tablename, indexname FROM pg_indexes WHERE schemaname = '%s'`, escapePGString(schemaName))
		if idxRows, err := db.Query(idxQuery); err == nil {
			idxMap := make(map[string][]string)
			for idxRows.Next() {
				var tName, iName string
				if err := idxRows.Scan(&tName, &iName); err == nil {
					idxMap[tName] = append(idxMap[tName], iName)
				}
			}
			idxRows.Close()

			for i, tbl := range results {
				if indexes, ok := idxMap[tbl.Name]; ok {
					results[i].Indexes = indexes
				}
			}
		}
	}

	return results, nil
}

// GetColumnsList returns all columns for a given table in a given schema.
func GetColumnsList(dbType string, db *sql.DB, schemaName string, tableName string) ([]ColumnInfo, error) {
	var results []ColumnInfo
	var query string
	var args []interface{}

	switch dbType {
	case "postgres":
		query = fmt.Sprintf(`SELECT column_name, data_type
				 FROM information_schema.columns
				 WHERE table_schema = '%s' AND table_name = '%s'
				 ORDER BY ordinal_position;`, escapePGString(schemaName), escapePGString(tableName))
	case "mysql":
		query = `SELECT COLUMN_NAME, COLUMN_TYPE
				 FROM information_schema.columns
				 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
				 ORDER BY ORDINAL_POSITION;`
		args = []interface{}{schemaName, tableName}
	case "sqlserver":
		query = `SELECT c.name AS column_name, typ.name AS data_type
				 FROM sys.objects t
				 JOIN sys.schemas s ON t.schema_id = s.schema_id
				 JOIN sys.columns c ON t.object_id = c.object_id
				 JOIN sys.types typ ON c.user_type_id = typ.user_type_id
				 WHERE s.name = @p1 AND t.name = @p2
				 ORDER BY c.column_id;`
		args = []interface{}{sql.Named("p1", schemaName), sql.Named("p2", tableName)}
	default:
		return nil, fmt.Errorf("unsupported database type for columns fetching: %s", dbType)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cName, cType sql.NullString
		if err := rows.Scan(&cName, &cType); err != nil {
			continue
		}
		if cName.Valid && cType.Valid {
			results = append(results, ColumnInfo{
				Name: cName.String,
				Type: cType.String,
			})
		}
	}

	return results, nil
}
