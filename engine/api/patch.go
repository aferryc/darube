package api

import (
	"encoding/json"
	"engine/store"
	"net/http"
)

// PatchConnectionFolderHandler handles PATCH /api/connections/{id}/folder
// It updates only the folder_id field of a connection configuration without attempting to reconnect.
func PatchConnectionFolderHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if id == "" {
        sendJSONResponse(w, CommandOutput{Success: false, Error: "id path parameter is required"}, http.StatusBadRequest)
        return
    }
    // Expect body like {"folder_id": "<folder-id>"}
    var payload struct {
        FolderID *string `json:"folder_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
        return
    }
    // Load existing config
    cfg, err := store.GetConnection(id)
    if err != nil {
        sendJSONResponse(w, CommandOutput{Success: false, Error: "Connection not found"}, http.StatusNotFound)
        return
    }
    if payload.FolderID != nil {
        cfg.FolderID = *payload.FolderID
    }
    if err := store.WriteConnection(*cfg); err != nil {
        sendJSONResponse(w, CommandOutput{Success: false, Error: "Failed to update connection"}, http.StatusInternalServerError)
        return
    }
    sendJSONResponse(w, CommandOutput{Success: true, Message: "Folder updated"}, http.StatusOK)
}
