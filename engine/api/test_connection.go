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
		// When connecting via Teleport local proxy, host/port are ignored and can be empty.
		if req.TeleportEnabled {
			if req.ConnectionName == "" || req.DBType == "" || req.User == "" || req.TeleportDBService == "" {
				sendJSONResponse(w, CommandOutput{
					Success: false,
					Error:   "Missing required fields (connection_name, db_type, user, teleport_db_service)",
				}, http.StatusBadRequest)
				return
			}
			if (req.DBType == "postgres" || req.DBType == "postgresql") && req.DBName == "" {
				sendJSONResponse(w, CommandOutput{
					Success: false,
					Error:   "Missing required field for Teleport Postgres (dbname)",
				}, http.StatusBadRequest)
				return
			}
			break
		}
		if req.ConnectionName == "" || req.DBType == "" || req.Host == "" || req.User == "" {
			sendJSONResponse(w, CommandOutput{
				Success: false,
				Error:   "Missing required fields (connection_name, db_type, host, user)",
			}, http.StatusBadRequest)
			return
		}
	}

	// Attempt connection purely for testing
	conn, cleanup, err := db.Connect(req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK) // 200 OK so frontend parses JSON error cleanly
		return
	}

	// Test successful, immediately close it
	conn.Close()
	if cleanup != nil {
		cleanup()
	}

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection successful!",
	}, http.StatusOK)
}
