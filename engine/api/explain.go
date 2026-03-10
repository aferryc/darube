package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"engine/store"
)

// ExplainHandler handles POST /api/connections/{id}/explain
func ExplainHandler(w http.ResponseWriter, r *http.Request) {
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

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Connection config not found"}, http.StatusInternalServerError)
		return
	}

	// Wrap query in EXPLAIN depending on DB Type
	var explainQuery string
	switch config.DBType {
	case "postgres", "postgresql":
		explainQuery = fmt.Sprintf("EXPLAIN (ANALYZE, COSTS, VERBOSE, BUFFERS, FORMAT JSON) %s", trimmedQuery)
	case "mysql", "mariadb":
		explainQuery = fmt.Sprintf("EXPLAIN FORMAT=JSON %s", trimmedQuery)
	case "sqlserver", "mssql":
		// SQL server JSON explain is not trivial in a single query block without SET SHOWPLAN_ALL
		// So we will fallback to standard text and parse minimally in frontend, or just return as text.
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Visual Explain is not yet supported for SQL Server"}, http.StatusNotImplemented)
		return
	default:
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Unsupported database type for EXPLAIN"}, http.StatusNotImplemented)
		return
	}

	// Execute Explain Query
	rows, err := conn.Query(explainQuery)
	if err != nil {
		sendJSONResponse(w, QueryResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "No plan returned"}, http.StatusInternalServerError)
		return
	}

	var planJSON string
	if config.DBType == "postgres" || config.DBType == "postgresql" {
		// Postgres usually returns a single column named "QUERY PLAN" containing the JSON string
		if err := rows.Scan(&planJSON); err != nil {
			sendJSONResponse(w, QueryResponse{Success: false, Error: "Failed to read Postgres plan: " + err.Error()}, http.StatusInternalServerError)
			return
		}
	} else if config.DBType == "mysql" || config.DBType == "mariadb" {
		// MySQL sometimes returns multiple columns or a single EXPLAIN column.
		// In FORMAT=JSON it returns a single column of JSON text.
		if err := rows.Scan(&planJSON); err != nil {
			sendJSONResponse(w, QueryResponse{Success: false, Error: "Failed to read MySQL plan: " + err.Error()}, http.StatusInternalServerError)
			return
		}
	}

	// Parse it as interface{} so it's clean JSON down the wire
	var rawPlan interface{}
	if err := json.Unmarshal([]byte(planJSON), &rawPlan); err != nil {
		sendJSONResponse(w, QueryResponse{Success: false, Error: "Failed to parse underlying JSON plan: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"plan":    rawPlan,
	}, http.StatusOK)
}
