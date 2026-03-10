package api

import (
	"net/http"

	"engine/store"
)

// DisconnectHandler handles POST /api/connections/{id}/disconnect
func DisconnectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	if !store.IsConnected(id) {
		sendJSONResponse(w, CommandOutput{
			Success: true,
			Message: "Connection is already disconnected",
		}, http.StatusOK)
		return
	}

	err := store.RemoveActiveConnection(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to close connection: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection disconnected",
	}, http.StatusOK)
}
