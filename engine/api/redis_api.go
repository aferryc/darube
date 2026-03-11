package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"engine/db"
	"engine/store"

	"github.com/google/uuid"
)

type RedisQueryRequest struct {
	Command string `json:"command"`
}

type RedisQueryResponse struct {
	Success bool            `json:"success"`
	Data    *db.RedisResult `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

var newRedisClient = db.NewRedisClient

// TestRedisHandler handles POST /api/redis/test
func TestRedisHandler(w http.ResponseWriter, r *http.Request) {
	var config store.RedisConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	client, err := newRedisClient(config)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK)
		return
	}
	client.Close()

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Connection successful",
	}, http.StatusOK)
}

// ConnectRedisHandler handles POST /api/redis and PUT /api/redis/{id}
func ConnectRedisHandler(w http.ResponseWriter, r *http.Request) {
	var config store.RedisConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	// ID from path takes precedence if it exists
	pathID := r.PathValue("id")
	if pathID != "" {
		config.ID = pathID
	}

	if config.ID == "" {
		config.ID = uuid.NewString()
	}

	// Preserve folder_id on updates unless explicitly changed via the folder PATCH endpoint.
	if pathID != "" && config.FolderID == "" {
		if existing, err := store.GetRedisConfig(pathID); err == nil && existing != nil {
			config.FolderID = existing.FolderID
		}
	}

	client, err := newRedisClient(config)
	if err != nil {
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   err.Error(),
		}, http.StatusOK)
		return
	}

	// Save to disk
	if err := store.WriteRedisConnection(config); err != nil {
		client.Close()
		sendJSONResponse(w, CommandOutput{
			Success: false,
			Error:   "Failed to save Redis config: " + err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	store.AddRedisConnection(config.ID, client)

	sendJSONResponse(w, CommandOutput{
		Success: true,
		Message: "Redis connected and saved",
		ID:      config.ID,
	}, http.StatusOK)
}

// ConnectSavedRedisHandler handles POST /api/redis/reconnect
func ConnectSavedRedisHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "Invalid request"}, http.StatusBadRequest)
		return
	}

	if store.IsRedisConnected(req.ID) {
		sendJSONResponse(w, CommandOutput{Success: true, Message: "Already connected"}, http.StatusOK)
		return
	}

	config, err := store.GetRedisConfig(req.ID)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusNotFound)
		return
	}

	client, err := newRedisClient(*config)
	if err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}

	store.AddRedisConnection(req.ID, client)

	sendJSONResponse(w, CommandOutput{Success: true, Message: "Redis connection established"}, http.StatusOK)
}

// RedisQueryHandler handles POST /api/redis/{id}/query
func RedisQueryHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, RedisQueryResponse{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}

	var req RedisQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, RedisQueryResponse{Success: false, Error: "Invalid request"}, http.StatusBadRequest)
		return
	}

	conn := store.GetRedisConnection(id)
	if conn == nil {
		sendJSONResponse(w, RedisQueryResponse{Success: false, Error: "Redis connection not active"}, http.StatusOK)
		return
	}

	redisClient := conn.(*db.RedisClient)
	
	// Performance hint: check for dangerous commands
	cmdUpper := strings.ToUpper(strings.TrimSpace(req.Command))
	if strings.HasPrefix(cmdUpper, "KEYS ") || cmdUpper == "KEYS *" || strings.HasPrefix(cmdUpper, "FLUSHALL") {
		// We could block or just let Execute return the warning. 
		// The user requested "Send a warning", so execution is allowed but flagged.
	}

	result, err := redisClient.Execute(r.Context(), req.Command)
	if err != nil {
		sendJSONResponse(w, RedisQueryResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}

	sendJSONResponse(w, RedisQueryResponse{
		Success: true,
		Data:    result,
	}, http.StatusOK)
}

// DeleteRedisConnectionHandler handles DELETE /api/redis/{id}
func DeleteRedisConnectionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, CommandOutput{Success: false, Error: "ID is required"}, http.StatusBadRequest)
		return
	}

	// 1. Unload from memory
	conn := store.GetRedisConnection(id)
	if conn != nil {
		redisClient := conn.(*db.RedisClient)
		redisClient.Close()
		store.RemoveRedisConnection(id)
	}

	// 2. Remove from disk
	if err := store.DeleteRedisConnection(id); err != nil {
		sendJSONResponse(w, CommandOutput{Success: false, Error: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, CommandOutput{Success: true, Message: "Redis connection deleted"}, http.StatusOK)
}

// DisconnectRedisHandler handles POST /api/redis/{id}/disconnect
func DisconnectRedisHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conn := store.GetRedisConnection(id)
	if conn != nil {
		redisClient := conn.(*db.RedisClient)
		redisClient.Close()
		store.RemoveRedisConnection(id)
	}

	sendJSONResponse(w, map[string]interface{}{"success": true}, http.StatusOK)
}
