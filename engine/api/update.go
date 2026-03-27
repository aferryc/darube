package api

import (
	"encoding/json"
	"net/http"

	"engine/db"
	"engine/store"
)

// UpdateConnectionHandler handles PUT /api/connections/{id}
func UpdateConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
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

	req.ID = id // Ensure ID matches path

	// If a password was not provided, keep the old one (if updating)
	oldConfig, err := store.GetConnection(id)
	if err == nil && req.Password == "" {
		req.Password = oldConfig.Password
	}

	// 1. Attempt connection with new credentials to verify
	conn, err := db.Connect(req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK)
		return
	}

	// 2. Save config to file (will overwrite since ID matches)
	err = store.WriteConnection(req)
	if err != nil {
		conn.Close()
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to update connection config: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// 3. Cache connection in memory (replacing old one)
	store.AddActiveConnection(req.ID, conn)
	startTableSizeEstimator(req.ID, req)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection updated and re-established",
	}, http.StatusOK)
}
