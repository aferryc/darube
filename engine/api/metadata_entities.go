package api

import (
	"net/http"

	"engine/metadata"
	"engine/store"
)

type MetadataEntitiesResponse struct {
	Success bool                  `json:"success"`
	Schemas []metadata.SchemaInfo `json:"schemas,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// GetMetadataEntitiesHandler handles GET /api/connections/{id}/metadata/entities
func GetMetadataEntitiesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, MetadataEntitiesResponse{
			Success: false,
			Error:   "id path parameter is required",
		}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, MetadataEntitiesResponse{
			Success: false,
			Error:   "Connection is not active. Please connect first.",
		}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, MetadataEntitiesResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusNotFound)
		return
	}

	entities, err := metadata.GetEntities(config.DBType, conn)
	if err != nil {
		sendJSONResponse(w, MetadataEntitiesResponse{
			Success: false,
			Error:   err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, MetadataEntitiesResponse{
		Success: true,
		Schemas: entities,
	}, http.StatusOK)
}
