package api

import (
	"encoding/json"
	"net/http"

	"engine/store"
)

// PatchRedisFolderHandler handles PATCH /api/redis/{id}/folder
// It updates only the folder_id field of a redis config without reconnecting.
func PatchRedisFolderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	var payload struct {
		FolderID *string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	cfg, err := store.GetRedisConfig(id)
	if err != nil || cfg == nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Redis connection not found"}, http.StatusNotFound)
		return
	}

	if payload.FolderID != nil {
		cfg.FolderID = *payload.FolderID
	}

	if err := store.WriteRedisConnection(*cfg); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Failed to update redis connection"}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, CommandOutput{Success: true, Message: "Folder updated"}, http.StatusOK)
}

