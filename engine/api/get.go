package api

import (
	"net/http"

	"engine/store"
)

// GetConnectionHandler handles GET /api/connections/{id}
func GetConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusNotFound)
		return
	}

	// Make a copy without the password for safe response
	safeConfig := *config
	safeConfig.Password = ""

	sendJSONResponse(w, safeConfig, http.StatusOK)
}
