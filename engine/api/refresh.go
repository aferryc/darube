package api

import (
	"net/http"

	"engine/db"
	"engine/store"
)

// RefreshConnectionHandler handles POST /api/connections/{id}/refresh
func RefreshConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	// 1. Unload existing connection
	err := store.RemoveActiveConnection(id)
	if err != nil {
		// Just log and continue if not found
		_ = err
	}

	// 2. Load latest config from file
	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to load connection settings: " + err.Error(),
		}, http.StatusNotFound)
		return
	}

	// 3. Reconnect
	conn, err := db.Connect(*config)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK) // Return 200 so frontend catches standard JSON formatting
		return
	}

	// 4. Stash back to memory
	store.AddActiveConnection(id, conn)
	startTableSizeEstimator(id, *config)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection refreshed successfully",
	}, http.StatusOK)
}
