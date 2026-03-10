package api

import (
	"encoding/json"
	"net/http"

	"engine/store"

	"github.com/google/uuid"
)

type FolderRequest struct {
	Name string `json:"name"`
}

// ListFoldersHandler handles GET /api/folders
func ListFoldersHandler(w http.ResponseWriter, r *http.Request) {
	folders, err := store.ReadFolders()
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"folders": folders,
	})
}

// CreateFolderHandler handles POST /api/folders
func CreateFolderHandler(w http.ResponseWriter, r *http.Request) {
	var req FolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "name is required"}, http.StatusBadRequest)
		return
	}

	folder := store.FolderConfig{
		ID:   uuid.NewString(),
		Name: req.Name,
	}

	if err := store.WriteFolder(folder); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"folder":  folder,
	}, http.StatusOK)
}

// UpdateFolderHandler handles PUT /api/folders/{id}
func UpdateFolderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "id required"}, http.StatusBadRequest)
		return
	}

	var req FolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "name is required"}, http.StatusBadRequest)
		return
	}

	folder := store.FolderConfig{ID: id, Name: req.Name}
	if err := store.WriteFolder(folder); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{"success": true, "folder": folder}, http.StatusOK)
}

// DeleteFolderHandler handles DELETE /api/folders/{id}
func DeleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "id required"}, http.StatusBadRequest)
		return
	}

	if err := store.DeleteFolder(id); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Clear folder_id from any connections that belonged to this folder
	conns, _ := store.ReadConnections()
	for _, c := range conns {
		if c.FolderID == id {
			c.FolderID = ""
			store.WriteConnection(c)
		}
	}

	// Clear folder_id from any redis connections that belonged to this folder
	redisConns, _ := store.ReadRedisConnections()
	for _, rc := range redisConns {
		if rc.FolderID == id {
			rc.FolderID = ""
			store.WriteRedisConnection(rc)
		}
	}

	sendJSONResponse(w, CommandOutput{Success: true, Message: "Folder deleted"}, http.StatusOK)
}
