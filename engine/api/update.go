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

	if !store.BeginConnect(req.ID) {
		sendJSONResponse(w, CommandOutput{
			Success: true,
			Message: "Connection attempt already in progress",
			ID:      req.ID,
		}, http.StatusOK)
		return
	}
	defer store.EndConnect(req.ID)

	// If a password was not provided, keep the old one (if updating)
	oldConfig, err := store.GetConnection(id)
	if err == nil && req.Password == "" {
		req.Password = oldConfig.Password
	}

	// 1. Attempt connection with new credentials to verify
	conn, cleanup, err := db.Connect(req)
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
		if cleanup != nil {
			cleanup()
		}
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to update connection config: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// 3. Cache connection in memory (replacing old one)
	store.AddActiveConnection(req.ID, conn)
	store.SetActiveCleanup(req.ID, cleanup)
	startTableSizeEstimator(req.ID, req)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection updated and re-established",
	}, http.StatusOK)
}
