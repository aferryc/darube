package api

import (
	"encoding/json"
	"net/http"

	"engine/db"
	"engine/store"
)

type ConnectSavedRequest struct {
	ID string `json:"id"`
}

// ConnectSavedHandler handles POST /api/connections/connect
func ConnectSavedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectSavedRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id is required",
		}, http.StatusBadRequest)
		return
	}

	if !store.BeginConnect(req.ID) {
		sendJSONResponse(w, CommandOutput{
			Success: true,
			Message: "Connection attempt already in progress",
		}, http.StatusOK)
		return
	}
	defer store.EndConnect(req.ID)

	// 1. Check if already connected
	if store.IsConnected(req.ID) {
		sendJSONResponse(w, CommandOutput{
			Success: true,
			Message: "Already connected",
		}, http.StatusOK)
		return
	}

	// 2. Load config from file
	config, err := store.GetConnection(req.ID)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusNotFound)
		return
	}

	// 3. Connect DB
	conn, cleanup, err := db.Connect(*config)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK)
		return
	}

	// 4. Trace in memory
	store.AddActiveConnection(req.ID, conn)
	store.SetActiveCleanup(req.ID, cleanup)
	startTableSizeEstimator(req.ID, *config)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection established",
	}, http.StatusOK)
}
