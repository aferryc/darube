package api

import (
	"net/http"

	"engine/metadata"
	"engine/store"
)

type DMLResponse struct {
	Success bool   `json:"success"`
	DML     string `json:"dml,omitempty"`
	Error   string `json:"error,omitempty"`
}

type IndexesResponse struct {
	Success bool                  `json:"success"`
	Indexes []metadata.IndexInfo  `json:"indexes,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// GetMetadataTableDMLHandler handles GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/dml
func GetMetadataTableDMLHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	schemaName := r.PathValue("schema")
	tableName := r.PathValue("table")

	if id == "" || schemaName == "" || tableName == "" {
		sendJSONResponse(w, DMLResponse{Success: false, Error: "Missing parameters"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, DMLResponse{Success: false, Error: "Connection is not active."}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, DMLResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	dml, err := metadata.GetTableDML(config.DBType, conn, schemaName, tableName)
	if err != nil {
		sendJSONResponse(w, DMLResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, DMLResponse{Success: true, DML: dml}, http.StatusOK)
}

// GetMetadataTableIndexesHandler handles GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/indexes
func GetMetadataTableIndexesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	schemaName := r.PathValue("schema")
	tableName := r.PathValue("table")

	if id == "" || schemaName == "" || tableName == "" {
		sendJSONResponse(w, IndexesResponse{Success: false, Error: "Missing parameters"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, IndexesResponse{Success: false, Error: "Connection is not active."}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, IndexesResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	indexes, err := metadata.GetTableIndexes(config.DBType, conn, schemaName, tableName)
	if err != nil {
		sendJSONResponse(w, IndexesResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, IndexesResponse{Success: true, Indexes: indexes}, http.StatusOK)
}
