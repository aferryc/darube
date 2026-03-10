package metadata

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetTableDML fetches the SHOW CREATE TABLE equivalent for the given table.
func GetTableDML(dbType string, db *sql.DB, schemaName string, tableName string) (string, error) {
	switch dbType {
	case "mysql":
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", schemaName, tableName)
		row := db.QueryRow(query)
		var tName, dml string
		err := row.Scan(&tName, &dml)
		if err != nil {
			return "", fmt.Errorf("failed to get DML for mysql table: %w", err)
		}
		return dml + ";", nil

	case "postgres":
		// Postgres doesn't have a simple SHOW CREATE TABLE. We will at least read the columns and constraints to build a basic one.
		// For a real production app, you'd use pg_dump or complex catalog queries. We will construct a basic representation.
		query := fmt.Sprintf(`
			SELECT column_name, data_type, character_maximum_length, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = '%s' AND table_name = '%s'
			ORDER BY ordinal_position;
		`, escapePGString(schemaName), escapePGString(tableName))

		rows, err := db.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		var columns []string
		for rows.Next() {
			var colName, dataType, isNullable sql.NullString
			var maxLen sql.NullInt64
			var colDef sql.NullString
			if err := rows.Scan(&colName, &dataType, &maxLen, &isNullable, &colDef); err != nil {
				continue
			}

			colStr := fmt.Sprintf("  %s %s", colName.String, dataType.String)
			if maxLen.Valid {
				colStr += fmt.Sprintf("(%d)", maxLen.Int64)
			}
			if isNullable.Valid && isNullable.String == "NO" {
				colStr += " NOT NULL"
			}
			if colDef.Valid {
				colStr += " DEFAULT " + colDef.String
			}
			columns = append(columns, colStr)
		}
		
		if len(columns) == 0 {
			return "", fmt.Errorf("table not found or has no columns")
		}

		dml := fmt.Sprintf("CREATE TABLE %s.%s (\n", schemaName, tableName)
		dml += strings.Join(columns, ",\n")
		dml += "\n);"
		return dml, nil

	case "sqlserver":
		return "-- DML generation for SQL Server is not fully implemented yet.\n-- Please use SSMS or ADS to script this table.", nil
		
	default:
		return "", fmt.Errorf("unsupported database type for DML: %s", dbType)
	}
}

// GetTableIndexes fetches index information for the given table.
func GetTableIndexes(dbType string, db *sql.DB, schemaName string, tableName string) ([]IndexInfo, error) {
	var results []IndexInfo
	
	switch dbType {
	case "mysql":
		query := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", schemaName, tableName)
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		idxMap := make(map[string]*IndexInfo)
		for rows.Next() {
			// mysql 8 might return 15 columns, mysql 5.7 might return 13. We use generic scan since show index varies.
			cols, _ := rows.Columns()
			vals := make([]interface{}, len(cols))
			for i := range vals {
				vals[i] = new(sql.RawBytes)
			}
			if err := rows.Scan(vals...); err != nil {
				continue
			}

			keyName := ""
			colName := ""
			nonUnique := "1"

			// map fields by index manually (usually: 0=Table, 1=Non_unique, 2=Key_name, 3=Seq_in_index, 4=Column_name)
			for i, col := range cols {
				val := string(*vals[i].(*sql.RawBytes))
				if col == "Key_name" { keyName = val }
				if col == "Column_name" { colName = val }
				if col == "Non_unique" { nonUnique = val }
			}

			if keyName == "" { continue }

			if _, ok := idxMap[keyName]; !ok {
				idxMap[keyName] = &IndexInfo{
					Name: keyName,
					Columns: []string{},
					Unique: nonUnique == "0",
					Primary: keyName == "PRIMARY",
				}
			}
			idxMap[keyName].Columns = append(idxMap[keyName].Columns, colName)
		}

		for _, idx := range idxMap {
			results = append(results, *idx)
		}
		
	case "postgres":
		query := fmt.Sprintf(`
			SELECT
				i.relname as index_name,
				a.attname as column_name,
				ix.indisunique as is_unique,
				ix.indisprimary as is_primary
			FROM
				pg_class t,
				pg_class i,
				pg_index ix,
				pg_attribute a,
				pg_namespace n
			WHERE
				t.oid = ix.indrelid
				and i.oid = ix.indexrelid
				and a.attrelid = t.oid
				and a.attnum = ANY(ix.indkey)
				and t.relkind = 'r'
				and n.oid = t.relnamespace
				and n.nspname = '%s'
				and t.relname = '%s'
			ORDER BY
				i.relname, a.attnum;
		`, escapePGString(schemaName), escapePGString(tableName))

		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		idxMap := make(map[string]*IndexInfo)
		for rows.Next() {
			var idxName, colName string
			var isUnique, isPrimary bool
			if err := rows.Scan(&idxName, &colName, &isUnique, &isPrimary); err != nil {
				continue
			}

			if _, ok := idxMap[idxName]; !ok {
				idxMap[idxName] = &IndexInfo{
					Name: idxName,
					Columns: []string{},
					Unique: isUnique,
					Primary: isPrimary,
				}
			}
			idxMap[idxName].Columns = append(idxMap[idxName].Columns, colName)
		}

		for _, idx := range idxMap {
			results = append(results, *idx)
		}

	case "sqlserver":
		return []IndexInfo{}, nil // Placeholder for SQL server
	}

	return results, nil
}
