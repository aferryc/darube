package api

import (
	"encoding/json"
	"net/http"

	"engine/db"
	"engine/store"
)

// TestConnectionHandler handles POST /api/connections/test
// It attempts to open a database connection using the provided credentials
// but immediately closes it without saving the configuration or keeping it active.
func TestConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req store.ConnectionConfig
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	if req.ConnectionName == "" || req.DBType == "" || req.Host == "" || req.User == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Missing required fields (connection_name, db_type, host, user)",
		}, http.StatusBadRequest)
		return
	}

	// Attempt connection purely for testing
	conn, err := db.Connect(req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK) // 200 OK so frontend parses JSON error cleanly
		return
	}

	// Test successful, immediately close it
	conn.Close()

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection successful!",
	}, http.StatusOK)
}
