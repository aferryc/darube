package api

import (
	"net/http"

	"engine/metadata"
	"engine/store"
)

type MetadataDatabasesResponse struct {
	Success   bool                    `json:"success"`
	Databases []metadata.DatabaseInfo `json:"databases,omitempty"`
	Error     string                  `json:"error,omitempty"`
}

// GetMetadataDatabasesHandler handles GET /api/connections/{id}/metadata/databases
func GetMetadataDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, MetadataDatabasesResponse{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	// Make sure we have the active connection
	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, MetadataDatabasesResponse{
			Success: false,
			Error:   "Connection is not active. Please connect first.",
		}, http.StatusBadRequest)
		return
	}

	// We need the config to know the db_type
	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, MetadataDatabasesResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusNotFound)
		return
	}

	dbs, err := metadata.GetDatabases(config.DBType, conn)
	if err != nil {
		sendJSONResponse(w, MetadataDatabasesResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, MetadataDatabasesResponse{
		Success:   true,
		Databases: dbs,
	}, http.StatusOK)
}
