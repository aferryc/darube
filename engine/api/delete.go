package api

import (
	"net/http"

	"engine/store"
)

// DeleteConnectionHandler handles DELETE /api/connections/{id}
func DeleteConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	// 1. Unload from memory, close socket
	err := store.RemoveActiveConnection(id)
	if err != nil {
		// Log but keep going to ensure it deletes from file
		_ = err
	}

	// 2. Erase from JSON registry
	err = store.DeleteConnection(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to delete config: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection deleted",
	}, http.StatusOK)
}
