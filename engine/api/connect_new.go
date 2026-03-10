package api

import (
	"encoding/json"
	"net/http"

	"engine/db"
	"engine/store"

	"github.com/google/uuid"
)

// ConnectNewHandler handles POST /api/connections
func ConnectNewHandler(w http.ResponseWriter, r *http.Request) {
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

	switch req.DBType {
	case "sqlite", "sqlite3":
		if req.ConnectionName == "" || req.DBType == "" || req.FilePath == "" {
			sendJSONResponse(w, CommandOutput{
				Success: false,
				Error:   "Missing required fields (connection_name, db_type, file_path)",
			}, http.StatusBadRequest)
			return
		}
	case "oracle":
		if req.ConnectionName == "" || req.DBType == "" || req.Host == "" || req.Port == 0 || req.User == "" || req.DBName == "" {
			sendJSONResponse(w, CommandOutput{
				Success: false,
				Error:   "Missing required fields (connection_name, db_type, host, port, user, dbname)",
			}, http.StatusBadRequest)
			return
		}
	default:
		if req.ConnectionName == "" || req.DBType == "" || req.Host == "" || req.User == "" {
			sendJSONResponse(w, CommandOutput{
				Success: false,
				Error:   "Missing required fields (connection_name, db_type, host, user)",
			}, http.StatusBadRequest)
			return
		}
	}

	if req.ID == "" {
		req.ID = uuid.NewString()
	}

	// 1. Attempt connection
	conn, err := db.Connect(req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK) // 200 OK so frontend parses JSON error
		return
	}

	// 2. Save config to file
	err = store.WriteConnection(req)
	if err != nil {
		conn.Close()
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to save connection config: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// 3. Cache connection in memory
	store.AddActiveConnection(req.ID, conn)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection successful and saved",
		ID:      req.ID,
	}, http.StatusOK)
}
