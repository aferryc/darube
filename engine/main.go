package main

import (
	"log"
	"net/http"
	"os"

	"engine/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	// List connections / Create a new connection
	mux.HandleFunc("GET /api/connections", api.ListConnectionsHandler)
	mux.HandleFunc("POST /api/connections", api.ConnectNewHandler)
	mux.HandleFunc("POST /api/connections/test", api.TestConnectionHandler)

	// Connect to an existing saved connection
	mux.HandleFunc("POST /api/connections/connect", api.ConnectSavedHandler)

	// Folder management
	mux.HandleFunc("GET /api/folders", api.ListFoldersHandler)
	mux.HandleFunc("POST /api/folders", api.CreateFolderHandler)
	mux.HandleFunc("PUT /api/folders/{id}", api.UpdateFolderHandler)
	mux.HandleFunc("DELETE /api/folders/{id}", api.DeleteFolderHandler)

	// Single connection CRUD & lifecycle (Go 1.22+ wildcard routing)
	mux.HandleFunc("GET /api/connections/{id}", api.GetConnectionHandler)
	mux.HandleFunc("PUT /api/connections/{id}", api.UpdateConnectionHandler)
	mux.HandleFunc("PATCH /api/connections/{id}/folder", api.PatchConnectionFolderHandler)
	mux.HandleFunc("DELETE /api/connections/{id}", api.DeleteConnectionHandler)

	// Extra controls
	mux.HandleFunc("POST /api/connections/{id}/disconnect", api.DisconnectHandler)
	mux.HandleFunc("POST /api/connections/{id}/refresh", api.RefreshConnectionHandler)
	
	// Query execution
	mux.HandleFunc("POST /api/connections/{id}/query", api.QueryHandler)
	mux.HandleFunc("POST /api/connections/{id}/mutate", api.MutateDataHandler)
	mux.HandleFunc("POST /api/connections/{id}/explain", api.ExplainHandler)

	// Redis Support (Separated)
	mux.HandleFunc("POST /api/redis/test", api.TestRedisHandler)
	mux.HandleFunc("POST /api/redis", api.ConnectRedisHandler)
	mux.HandleFunc("PUT /api/redis/{id}", api.ConnectRedisHandler)
	mux.HandleFunc("DELETE /api/redis/{id}", api.DeleteRedisConnectionHandler)
		mux.HandleFunc("POST /api/redis/reconnect", api.ConnectSavedRedisHandler)
		mux.HandleFunc("POST /api/redis/{id}/query", api.RedisQueryHandler)
		mux.HandleFunc("POST /api/redis/{id}/disconnect", api.DisconnectRedisHandler)
		mux.HandleFunc("POST /api/redis/{id}/export", api.RedisExportHandler)
		mux.HandleFunc("PATCH /api/redis/{id}/folder", api.PatchRedisFolderHandler)

	// Data Export
	mux.HandleFunc("POST /api/connections/{id}/export", api.ExportHandler)
	mux.HandleFunc("POST /api/scripts/run", api.RunScriptHandler)

	// Metadata controls
	mux.HandleFunc("GET /api/connections/{id}/metadata/databases", api.GetMetadataDatabasesHandler)
	mux.HandleFunc("GET /api/connections/{id}/metadata/entities", api.GetMetadataEntitiesHandler) // Keeping old route for compatibility
	
	// Lazy Metadata Controls
	mux.HandleFunc("GET /api/connections/{id}/metadata/schemas", api.GetMetadataSchemasHandler)
	mux.HandleFunc("GET /api/connections/{id}/metadata/schemas/{schema}/tables", api.GetMetadataTablesHandler)
	mux.HandleFunc("GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/columns", api.GetMetadataColumnsHandler)
	mux.HandleFunc("GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/dml", api.GetMetadataTableDMLHandler)
	mux.HandleFunc("GET /api/connections/{id}/metadata/schemas/{schema}/tables/{table}/indexes", api.GetMetadataTableIndexesHandler)

	// Workspace state
	mux.HandleFunc("GET /api/workspace", api.GetWorkspaceHandler)
	mux.HandleFunc("POST /api/workspace", api.SaveWorkspaceHandler)

	log.Printf("Starting Darube DB Engine on port %s", port)
	if err := http.ListenAndServe(":"+port, enableCORS(mux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// enableCORS is a simple middleware to allow cross-origin requests from the Electron/Vite frontend
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Or explicitly "http://localhost:5173"
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
