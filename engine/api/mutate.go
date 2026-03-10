package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"engine/store"
)

type MutationAction struct {
	Type        string                 `json:"type"` // "update", "delete", "insert"
	OriginalRow map[string]interface{} `json:"original_row,omitempty"`
	NewValues   map[string]interface{} `json:"new_values,omitempty"`
}

type MutateRequest struct {
	Table     string           `json:"table"`
	Mutations []MutationAction `json:"mutations"`
}

type MutateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MutateDataHandler handles POST /api/connections/{id}/mutate
func MutateDataHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, MutateResponse{Success: false, Error: "Connection ID is required"}, http.StatusBadRequest)
		return
	}

	var req MutateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, MutateResponse{Success: false, Error: "Invalid JSON payload"}, http.StatusBadRequest)
		return
	}

	if req.Table == "" || len(req.Mutations) == 0 {
		sendJSONResponse(w, MutateResponse{Success: false, Error: "Table name and at least one mutation are required"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, MutateResponse{Success: false, Error: "Connection is not active"}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, MutateResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	// Begin Transaction
	ctx := context.Background()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		sendJSONResponse(w, MutateResponse{Success: false, Error: "Failed to begin transaction: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	// Make sure we either commit or rollback
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for i, mut := range req.Mutations {
		switch mut.Type {
		case "update":
			err = executeUpdate(tx, config.DBType, req.Table, mut.OriginalRow, mut.NewValues)
		case "delete":
			err = executeDelete(tx, config.DBType, req.Table, mut.OriginalRow)
		case "insert":
			err = executeInsert(tx, config.DBType, req.Table, mut.NewValues)
		default:
			err = fmt.Errorf("unknown mutation type: %s", mut.Type)
		}

		if err != nil {
			err = fmt.Errorf("mutation %d failed: %w", i+1, err)
			sendJSONResponse(w, MutateResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
			return // this will trigger the defer rollback
		}
	}

	sendJSONResponse(w, MutateResponse{Success: true, Message: fmt.Sprintf("Successfully applied %d mutations.", len(req.Mutations))}, http.StatusOK)
}

func placeholder(dbType string, idx int) string {
	switch dbType {
	case "postgres":
		return fmt.Sprintf("$%d", idx)
	case "sqlserver":
		return fmt.Sprintf("@p%d", idx)
	default:
		return "?"
	}
}

func executeUpdate(tx *sql.Tx, dbType string, table string, old map[string]interface{}, new map[string]interface{}) error {
	if len(new) == 0 {
		return nil
	}

	var setCols []string
	var args []interface{}
	argIdx := 1

	for k, v := range new {
		setCols = append(setCols, fmt.Sprintf("%s = %s", quoteIdent(dbType, k), placeholder(dbType, argIdx)))
		args = append(args, v)
		argIdx++
	}

	whereClause, whereArgs, err := buildWhereClause(dbType, old, &argIdx)
	if err != nil {
		return err
	}
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, strings.Join(setCols, ", "), whereClause)

	res, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("update affected 0 rows (optimistic concurrency failure)")
	}
	return nil
}

func executeDelete(tx *sql.Tx, dbType string, table string, old map[string]interface{}) error {
	argIdx := 1
	whereClause, whereArgs, err := buildWhereClause(dbType, old, &argIdx)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, whereClause)
	res, err := tx.Exec(query, whereArgs...)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("delete affected 0 rows (row might have been changed/deleted already)")
	}
	return nil
}

func executeInsert(tx *sql.Tx, dbType string, table string, newVals map[string]interface{}) error {
	if len(newVals) == 0 {
		return fmt.Errorf("cannot insert empty values")
	}

	var cols []string
	var placeholders []string
	var args []interface{}
	argIdx := 1

	for k, v := range newVals {
		cols = append(cols, quoteIdent(dbType, k))
		placeholders = append(placeholders, placeholder(dbType, argIdx))
		args = append(args, v)
		argIdx++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	_, err := tx.Exec(query, args...)
	return err
}

func buildWhereClause(dbType string, old map[string]interface{}, argIdx *int) (string, []interface{}, error) {
	if len(old) == 0 {
		return "", nil, fmt.Errorf("cannot perform operation without old row values for WHERE clause")
	}

	var conditions []string
	var args []interface{}

	for k, v := range old {
		if v == nil {
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", quoteIdent(dbType, k)))
		} else {
			conditions = append(conditions, fmt.Sprintf("%s = %s", quoteIdent(dbType, k), placeholder(dbType, *argIdx)))
			args = append(args, v)
			*argIdx++
		}
	}

	return strings.Join(conditions, " AND "), args, nil
}

func quoteIdent(dbType, ident string) string {
	switch dbType {
	case "mysql":
		return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
	case "sqlserver":
		return "[" + strings.ReplaceAll(ident, "]", "]]") + "]"
	default:
		// Postgres
		return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
	}
}
