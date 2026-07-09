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

	// Teleport metadata so the UI can prompt for `tsh login` in-app before
	// connecting instead of surfacing the engine's raw "run tsh login" error.
	TeleportEnabled   bool   `json:"teleport_enabled,omitempty"`
	TeleportProfileID string `json:"teleport_profile_id,omitempty"`
	TeleportProfile   string `json:"teleport_profile,omitempty"`
	TeleportUser      string `json:"teleport_user,omitempty"`
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
			ID:                c.ID,
			ConnectionName:    c.ConnectionName,
			DBType:            c.DBType,
			Host:              c.Host,
			Port:              c.Port,
			User:              c.User,
			Status:            status,
			FolderID:          c.FolderID,
			TeleportEnabled:   c.TeleportEnabled,
			TeleportProfileID: c.TeleportProfileID,
			TeleportProfile:   c.TeleportProfile,
			TeleportUser:      c.TeleportUser,
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

	// 3. HTTP Connections
	httpSaved, err := store.ReadHTTPConnections()
	if err == nil {
		for _, c := range httpSaved {
			results = append(results, ConnectionStatus{
				ID:             c.ID,
				ConnectionName: c.ConnectionName,
				DBType:         "http",
				Host:           c.BaseURL,
				Port:           0,
				User:           c.Auth.Username,
				Status:         "connected", // stateless
				FolderID:       c.FolderID,
			})
		}
	}

	// 4. gRPC Connections
	grpcSaved, err := store.ReadGRPCConnections()
	if err == nil {
		for _, c := range grpcSaved {
			results = append(results, ConnectionStatus{
				ID:             c.ID,
				ConnectionName: c.ConnectionName,
				DBType:         "grpc",
				Host:           c.Address,
				Port:           0,
				User:           c.Auth.Username,
				Status:         "connected", // stateless
				FolderID:       c.FolderID,
			})
		}
	}

	sendJSONResponse(w, ListConnectionsResponse{
		Connections: results,
	}, http.StatusOK)
}
