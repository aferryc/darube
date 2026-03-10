package api

import (
	"encoding/json"
	"net/http"

	"engine/store"
)

// GetWorkspaceHandler handles GET /api/workspace
func GetWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := store.LoadWorkspace()
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, ws, http.StatusOK)
}

// SaveWorkspaceHandler handles POST /api/workspace
func SaveWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	var ws store.WorkspaceState

	if err := json.NewDecoder(r.Body).Decode(&ws); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Invalid payload format"}, http.StatusBadRequest)
		return
	}

	if err := store.SaveWorkspace(ws); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Failed to save workspace"}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{"success": true, "message": "Workspace saved"}, http.StatusOK)
}
