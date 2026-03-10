package metadata

import (
	"database/sql"
	"fmt"
)

// GetEntities executes a driver-specific query to list tables, views, columns, and indexes grouped by schema.
func GetEntities(dbType string, db *sql.DB) ([]SchemaInfo, error) {
	var results []SchemaInfo

	switch dbType {
	case "postgres":
		query := `SELECT t.table_schema, t.table_name, t.table_type, c.column_name, c.data_type
				  FROM information_schema.tables t
				  LEFT JOIN information_schema.columns c 
				  ON t.table_schema = c.table_schema AND t.table_name = c.table_name
				  WHERE t.table_schema NOT IN ('information_schema', 'pg_catalog')
				  ORDER BY t.table_schema, t.table_name, c.ordinal_position`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		schemaMap := make(map[string]*SchemaInfo)
		tableMap := make(map[string]map[string]*EntityInfo)

		for rows.Next() {
			var schemaName, tableName string
			var pType, colName, colType sql.NullString
			if err := rows.Scan(&schemaName, &tableName, &pType, &colName, &colType); err != nil {
				fmt.Printf("Row scan error: %v\n", err)
				continue
			}

			if _, ok := schemaMap[schemaName]; !ok {
				schemaMap[schemaName] = &SchemaInfo{Name: schemaName, Tables: []EntityInfo{}}
				tableMap[schemaName] = make(map[string]*EntityInfo)
			}

			if _, ok := tableMap[schemaName][tableName]; !ok {
				objType := "table"
				if pType.Valid && pType.String == "VIEW" {
					objType = "view"
				}
				tableMap[schemaName][tableName] = &EntityInfo{
					Name:    tableName,
					Type:    objType,
					Columns: []ColumnInfo{},
					Indexes: []string{},
				}
			}

			if colName.Valid && colType.Valid {
				tableMap[schemaName][tableName].Columns = append(tableMap[schemaName][tableName].Columns, ColumnInfo{
					Name: colName.String,
					Type: colType.String,
				})
			}
		}

		// Get indexes
		idxQuery := `SELECT schemaname, tablename, indexname FROM pg_indexes WHERE schemaname NOT IN ('information_schema', 'pg_catalog')`
		if idxRows, err := db.Query(idxQuery); err == nil {
			for idxRows.Next() {
				var sName, tName, iName string
				if err := idxRows.Scan(&sName, &tName, &iName); err == nil {
					if schema, ok := tableMap[sName]; ok {
						if tbl, ok := schema[tName]; ok {
							tbl.Indexes = append(tbl.Indexes, iName)
						}
					}
				}
			}
			idxRows.Close()
		}

		for sName, sInfo := range schemaMap {
			for _, tblInfo := range tableMap[sName] {
				sInfo.Tables = append(sInfo.Tables, *tblInfo)
			}
			results = append(results, *sInfo)
		}

	case "mysql":
		query := `SELECT t.TABLE_SCHEMA, t.TABLE_NAME, t.TABLE_TYPE, c.COLUMN_NAME, c.COLUMN_TYPE
				  FROM information_schema.tables t
				  LEFT JOIN information_schema.columns c 
				  ON t.TABLE_SCHEMA = c.TABLE_SCHEMA AND t.TABLE_NAME = c.TABLE_NAME
				  WHERE t.TABLE_SCHEMA = DATABASE()
				  ORDER BY t.TABLE_NAME, c.ORDINAL_POSITION`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		schemaMap := make(map[string]*SchemaInfo)
		tableMap := make(map[string]map[string]*EntityInfo)

		for rows.Next() {
			var schemaName, tableName, pType string
			var colName, colType sql.NullString
			if err := rows.Scan(&schemaName, &tableName, &pType, &colName, &colType); err != nil {
				continue
			}

			if _, ok := schemaMap[schemaName]; !ok {
				schemaMap[schemaName] = &SchemaInfo{Name: schemaName, Tables: []EntityInfo{}}
				tableMap[schemaName] = make(map[string]*EntityInfo)
			}

			if _, ok := tableMap[schemaName][tableName]; !ok {
				objType := "table"
				if pType == "VIEW" {
					objType = "view"
				}
				tableMap[schemaName][tableName] = &EntityInfo{
					Name:    tableName,
					Type:    objType,
					Columns: []ColumnInfo{},
					Indexes: []string{},
				}
			}

			if colName.Valid && colType.Valid {
				tableMap[schemaName][tableName].Columns = append(tableMap[schemaName][tableName].Columns, ColumnInfo{
					Name: colName.String,
					Type: colType.String,
				})
			}
		}

		// Group MySQL tables into schemas
		for sName, sInfo := range schemaMap {
			for _, tblInfo := range tableMap[sName] {
				sInfo.Tables = append(sInfo.Tables, *tblInfo)
			}
			results = append(results, *sInfo)
		}

	case "sqlserver":
		query := `SELECT s.name AS schema_name, t.name AS table_name, t.type_desc AS table_type, c.name AS column_name, typ.name AS data_type
				  FROM sys.objects t
				  JOIN sys.schemas s ON t.schema_id = s.schema_id
				  LEFT JOIN sys.columns c ON t.object_id = c.object_id
				  LEFT JOIN sys.types typ ON c.user_type_id = typ.user_type_id
				  WHERE t.type IN ('U', 'V')
				  ORDER BY s.name, t.name, c.column_id`
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		schemaMap := make(map[string]*SchemaInfo)
		tableMap := make(map[string]map[string]*EntityInfo)

		for rows.Next() {
			var schemaName, tableName, pType string
			var colName, colType sql.NullString
			if err := rows.Scan(&schemaName, &tableName, &pType, &colName, &colType); err != nil {
				continue
			}

			if _, ok := schemaMap[schemaName]; !ok {
				schemaMap[schemaName] = &SchemaInfo{Name: schemaName, Tables: []EntityInfo{}}
				tableMap[schemaName] = make(map[string]*EntityInfo)
			}

			if _, ok := tableMap[schemaName][tableName]; !ok {
				objType := "table"
				if pType == "VIEW" {
					objType = "view"
				}
				tableMap[schemaName][tableName] = &EntityInfo{
					Name:    tableName,
					Type:    objType,
					Columns: []ColumnInfo{},
					Indexes: []string{},
				}
			}

			if colName.Valid && colType.Valid {
				tableMap[schemaName][tableName].Columns = append(tableMap[schemaName][tableName].Columns, ColumnInfo{
					Name: colName.String,
					Type: colType.String,
				})
			}
		}

		for sName, sInfo := range schemaMap {
			for _, tblInfo := range tableMap[sName] {
				sInfo.Tables = append(sInfo.Tables, *tblInfo)
			}
			results = append(results, *sInfo)
		}

	default:
		return nil, fmt.Errorf("unsupported database type for metadata fetching: %s", dbType)
	}

	return results, nil
}
