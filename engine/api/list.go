package api

import (
	"net/http"

	"engine/store"
)

type ConnectionStatus struct {
	ID             string `json:"id"`
	ConnectionName string `json:"connection_name"`
	DBType         string `json:"db_type"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Status         string `json:"status"` // "connected" or "disconnected"
	FolderID       string `json:"folder_id,omitempty"`
}

type ListConnectionsResponse struct {
	Connections []ConnectionStatus `json:"connections"`
}

// ListConnectionsHandler handles GET /api/connections
func ListConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. SQL Connections
	saved, err := store.ReadConnections()
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	var results []ConnectionStatus
	for _, c := range saved {
		status := "disconnected"
		if store.IsConnected(c.ID) {
			status = "connected"
		}
		results = append(results, ConnectionStatus{
			ID:             c.ID,
			ConnectionName: c.ConnectionName,
			DBType:         c.DBType,
			Host:           c.Host,
			Port:           c.Port,
			User:           c.User,
			Status:         status,
			FolderID:       c.FolderID,
		})
	}

	// 2. Redis Connections
	redisSaved, err := store.ReadRedisConnections()
	if err == nil { // Silently ignore if we can't read redis connections for now, or log it
		for _, c := range redisSaved {
			status := "disconnected"
			if store.IsRedisConnected(c.ID) {
				status = "connected"
			}
			results = append(results, ConnectionStatus{
				ID:             c.ID,
				ConnectionName: c.ConnectionName,
				DBType:         "redis",
				Host:           c.Host,
				Port:           c.Port,
				User:           c.User,
				Status:         status,
				FolderID:       c.FolderID,
			})
		}
	}

	sendJSONResponse(w, ListConnectionsResponse{
		Connections: results,
	}, http.StatusOK)
}
