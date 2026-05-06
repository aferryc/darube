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
		"success":                  true,
		"layout_direction":         s.LayoutDirection,
		"teleport_profiles":        s.TeleportProfiles,
		"global_script_timeout_ms": s.GlobalScriptTimeoutMs,
		"global_query_timeout_ms":  s.GlobalQueryTimeoutMs,
		"global_api_timeout_ms":    s.GlobalAPITimeoutMs,
		"max_lines_query":          s.MaxLinesQuery,
		"max_lines_script":         s.MaxLinesScript,
		"max_lines_body":           s.MaxLinesBody,
		"large_result_warn_mb":     s.LargeResultWarnMB,
		"theme_variant":            s.ThemeVariant,
		"ui_theme_custom":          s.UIThemeCustom,
		"ui_font_family":           s.UIFontFamily,
		"ui_font_size":             s.UIFontSize,
		"ui_font_color":            s.UIFontColor,
		"ui_text_primary":          s.UITextPrimary,
		"ui_text_muted":            s.UITextMuted,
		"ui_text_accent":           s.UITextAccent,
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

	// Global timeouts – accept numbers; negative means "no limit".
	if v, ok := patch["global_script_timeout_ms"]; ok {
		switch n := v.(type) {
		case float64:
			s.GlobalScriptTimeoutMs = int(n)
		case int:
			s.GlobalScriptTimeoutMs = n
		}
	}
	if v, ok := patch["global_query_timeout_ms"]; ok {
		switch n := v.(type) {
		case float64:
			s.GlobalQueryTimeoutMs = int(n)
		case int:
			s.GlobalQueryTimeoutMs = n
		}
	}
	if v, ok := patch["global_api_timeout_ms"]; ok {
		switch n := v.(type) {
		case float64:
			s.GlobalAPITimeoutMs = int(n)
		case int:
			s.GlobalAPITimeoutMs = n
		}
	}

	// Max lines.
	if v, ok := patch["max_lines_query"]; ok {
		switch n := v.(type) {
		case float64:
			s.MaxLinesQuery = int(n)
		case int:
			s.MaxLinesQuery = n
		}
	}
	if v, ok := patch["max_lines_script"]; ok {
		switch n := v.(type) {
		case float64:
			s.MaxLinesScript = int(n)
		case int:
			s.MaxLinesScript = n
		}
	}
	if v, ok := patch["max_lines_body"]; ok {
		switch n := v.(type) {
		case float64:
			s.MaxLinesBody = int(n)
		case int:
			s.MaxLinesBody = n
		}
	}
	if v, ok := patch["large_result_warn_mb"]; ok {
		switch n := v.(type) {
		case float64:
			s.LargeResultWarnMB = int(n)
		case int:
			s.LargeResultWarnMB = n
		}
	}

	// Theme + typography.
	if v, ok := patch["theme_variant"]; ok {
		if str, ok := v.(string); ok {
			s.ThemeVariant = strings.TrimSpace(str)
		}
	}
	if v, ok := patch["ui_theme_custom"]; ok {
		if str, ok := v.(string); ok {
			s.UIThemeCustom = str
		}
	}
	if v, ok := patch["ui_font_family"]; ok {
		if str, ok := v.(string); ok {
			s.UIFontFamily = strings.TrimSpace(str)
		}
	}
	if v, ok := patch["ui_font_size"]; ok {
		switch n := v.(type) {
		case float64:
			s.UIFontSize = int(n)
		case int:
			s.UIFontSize = n
		}
	}
	if v, ok := patch["ui_font_color"]; ok {
		if str, ok := v.(string); ok {
			s.UIFontColor = strings.TrimSpace(str)
		}
	}
	if v, ok := patch["ui_text_primary"]; ok {
		if str, ok := v.(string); ok {
			s.UITextPrimary = strings.TrimSpace(str)
		}
	}
	if v, ok := patch["ui_text_muted"]; ok {
		if str, ok := v.(string); ok {
			s.UITextMuted = strings.TrimSpace(str)
		}
	}
	if v, ok := patch["ui_text_accent"]; ok {
		if str, ok := v.(string); ok {
			s.UITextAccent = strings.TrimSpace(str)
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

// TeleportStatusHandler handles GET /api/teleport/status
// Optional query param: ?profile=<name> to inspect a specific tsh profile.
// Returns a UI-friendly equivalent of `tsh status`.
func TeleportStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profileName := r.URL.Query().Get("profile")
	st := teleport.Status(profileName)
	sendJSONResponse(w, map[string]interface{}{
		"success":       true,
		"tsh_available": st.TSHAvailable,
		"tsh_path":      st.TSHPath,
		"logged_in":     st.LoggedIn,
		"cluster":       st.Cluster,
		"user":          st.User,
		"profile":       st.Profile,
		"proxy":         st.Proxy,
		"expires_at":    st.ExpiresAt,
		"error":         st.Error,
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

// TeleportLoginHandler handles POST /api/teleport/login
// Body: {"proxy": "teleport.example.com"}
// Runs `tsh login --proxy=<host>` which opens the system browser for SSO.
func TeleportLoginHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Proxy string `json:"proxy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Proxy == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "proxy is required"}, http.StatusBadRequest)
		return
	}
	result, err := teleport.Login(r.Context(), body.Proxy)
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error(), "output": result.Output}, http.StatusOK)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "output": result.Output}, http.StatusOK)
}

// TeleportListDatabasesHandler handles GET /api/teleport/databases
// Optional query param: ?profile=<name> to use a specific tsh profile.
func TeleportListDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	profileName := r.URL.Query().Get("profile")
	dbs, err := teleport.ListDatabases(r.Context(), profileName)
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": err.Error()}, http.StatusOK)
		return
	}
	sendJSONResponse(w, map[string]interface{}{"success": true, "databases": dbs}, http.StatusOK)
}
