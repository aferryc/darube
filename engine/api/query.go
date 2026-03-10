package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"engine/store"
)

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Success      bool            `json:"success"`
	Columns      []string        `json:"columns,omitempty"`
	Rows         [][]interface{} `json:"rows,omitempty"`
	RowsAffected int64           `json:"rows_affected,omitempty"`
	Message      string          `json:"message,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// QueryHandler handles POST /api/connections/{id}/query
func QueryHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	trimmedQuery := strings.TrimSpace(req.Query)
	if trimmedQuery == "" {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Query cannot be empty"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Connection is not active"}, http.StatusBadRequest)
		return
	}

	// Very basic heuristic to check if query returns rows
	upperQuery := strings.ToUpper(trimmedQuery)
	returnsRows := strings.HasPrefix(upperQuery, "SELECT") ||
		strings.HasPrefix(upperQuery, "SHOW") ||
		strings.HasPrefix(upperQuery, "EXPLAIN") ||
		strings.HasPrefix(upperQuery, "DESCRIBE") ||
		strings.HasPrefix(upperQuery, "PRAGMA")

	if returnsRows {
		handleSelectQuery(w, conn, trimmedQuery)
	} else {
		handleMutationQuery(w, conn, trimmedQuery)
	}
}

func handleMutationQuery(w http.ResponseWriter, db *sql.DB, query string) {
	result, err := db.Exec(query)
	if err != nil {
		sendJSONResponse(w, QueryResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK) // 200 OK so frontend receives the JSON error payload
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// some drivers don't support RowsAffected, that's okay
		affected = 0
	}

	sendJSONResponse(w, QueryResponse{
		Success:      true,
		RowsAffected: affected,
		Message:      fmt.Sprintf("%d rows affected", affected),
	}, http.StatusOK)
}

func handleSelectQuery(w http.ResponseWriter, db *sql.DB, query string) {
	rows, err := db.Query(query)
	if err != nil {
		sendJSONResponse(w, QueryResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		sendJSONResponse(w, QueryResponse{
			Success: false,
			Error:   "Failed to read columns: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// Prepare dynamic container
	var resultRows [][]interface{}

	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the first slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			sendJSONResponse(w, QueryResponse{
				Success: false,
				Error:   "Failed to scan row: " + err.Error(),
			}, http.StatusInternalServerError)
			return
		}

		// Create row copy
		var rowData []interface{}
		for _, colPtr := range columnPointers {
			val := *(colPtr.(*interface{}))

			// Handle byte slices safely by converting to string
			b, ok := val.([]byte)
			if ok {
				val = string(b)
			}
			
			// Some drivers (like MySQL) return int64, others return float64, etc. JSON marshalling handles it natively.

			rowData = append(rowData, val)
		}

		resultRows = append(resultRows, rowData)
	}

	if err = rows.Err(); err != nil {
		sendJSONResponse(w, QueryResponse{
			Success: false,
			Error:   "Error iterating rows: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	if resultRows == nil {
		resultRows = make([][]interface{}, 0) // Prevents `null` in JSON output, uses `[]`
	}

	sendJSONResponse(w, QueryResponse{
		Success: true,
		Columns: cols,
		Rows:    resultRows,
		Message: "OK",
	}, http.StatusOK)
}
