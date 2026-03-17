package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"engine/store"
	"engine/teleport"
)

// GetSettingsHandler handles GET /api/settings
func GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	s, err := store.LoadSettings()
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, map[string]interface{}{
		"success":           true,
		"layout_direction":  s.LayoutDirection,
		"teleport_profiles": s.TeleportProfiles,
	}, http.StatusOK)
}

// PutSettingsHandler handles PUT /api/settings
func PutSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s, err := store.LoadSettings()
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Invalid JSON"}, http.StatusBadRequest)
		return
	}
	if v, ok := patch["layout_direction"]; ok {
		if str, ok := v.(string); ok && strings.TrimSpace(str) != "" {
			if str == "vertical" || str == "horizontal" {
				s.LayoutDirection = str
			}
		}
	}
	if err := store.SaveSettings(s); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "message": "Settings saved"}, http.StatusOK)
}

// CreateTeleportProfileHandler handles POST /api/settings/teleport-profiles
func CreateTeleportProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p store.TeleportProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Invalid JSON"}, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(p.Name) == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "name is required"}, http.StatusBadRequest)
		return
	}
	created, err := store.CreateTeleportProfile(p)
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "profile": created}, http.StatusOK)
}

// UpdateTeleportProfileHandler handles PUT /api/settings/teleport-profiles/{id}
func UpdateTeleportProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "ID is required"}, http.StatusBadRequest)
		return
	}
	var p store.TeleportProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Invalid JSON"}, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(p.Name) == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "name is required"}, http.StatusBadRequest)
		return
	}
	if err := store.UpdateTeleportProfile(id, p); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "message": "Profile updated"}, http.StatusOK)
}

// TeleportDetectHandler handles GET /api/teleport/detect
func TeleportDetectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p, err := teleport.DetectTSHProfile()
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusOK)
		return
	}
	sendJSONResponse(w, map[string]interface{}{
		"success": true,
		"cluster": p.Cluster,
		"user":    p.User,
		"profile": p.Profile,
	}, http.StatusOK)
}

// DeleteTeleportProfileHandler handles DELETE /api/settings/teleport-profiles/{id}
func DeleteTeleportProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "ID is required"}, http.StatusBadRequest)
		return
	}
	if err := store.DeleteTeleportProfile(id); err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "message": "Profile deleted"}, http.StatusOK)
}
