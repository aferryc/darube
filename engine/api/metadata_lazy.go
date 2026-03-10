package api

import (
	"net/http"

	"engine/metadata"
	"engine/store"
)

type LazySchemasResponse struct {
	Success bool                  `json:"success"`
	Schemas []metadata.SchemaInfo `json:"schemas,omitempty"`
	Error   string                `json:"error,omitempty"`
}

type LazyTablesResponse struct {
	Success bool                  `json:"success"`
	Tables  []metadata.EntityInfo `json:"tables,omitempty"`
	Error   string                `json:"error,omitempty"`
}

type LazyColumnsResponse struct {
	Success bool                  `json:"success"`
	Columns []metadata.ColumnInfo `json:"columns,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// GetMetadataSchemasHandler handles GET /api/connections/{id}/metadata/schemas
func GetMetadataSchemasHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, LazySchemasResponse{Success: false, Error: "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, LazySchemasResponse{Success: false, Error: "Connection is not active."}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, LazySchemasResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	schemas, err := metadata.GetSchemas(config.DBType, conn)
	if err != nil {
		sendJSONResponse(w, LazySchemasResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, LazySchemasResponse{Success: true, Schemas: schemas}, http.StatusOK)
}

// GetMetadataTablesHandler handles GET /api/connections/{id}/metadata/schemas/{schema}/tables
func GetMetadataTablesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	schemaName := r.PathValue("schema")
	
	if id == "" || schemaName == "" {
		sendJSONResponse(w, LazyTablesResponse{Success: false, Error: "Missing parameters"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, LazyTablesResponse{Success: false, Error: "Connection is not active."}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, LazyTablesResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	tables, err := metadata.GetTablesList(config.DBType, conn, schemaName)
	if err != nil {
		sendJSONResponse(w, LazyTablesResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, LazyTablesResponse{Success: true, Tables: tables}, http.StatusOK)
}

// GetMetadataColumnsHandler handles GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/columns
func GetMetadataColumnsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	schemaName := r.PathValue("schema")
	tableName := r.PathValue("table")
	
	if id == "" || schemaName == "" || tableName == "" {
		sendJSONResponse(w, LazyColumnsResponse{Success: false, Error: "Missing parameters"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, LazyColumnsResponse{Success: false, Error: "Connection is not active."}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, LazyColumnsResponse{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	columns, err := metadata.GetColumnsList(config.DBType, conn, schemaName, tableName)
	if err != nil {
		sendJSONResponse(w, LazyColumnsResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, LazyColumnsResponse{Success: true, Columns: columns}, http.StatusOK)
}
