package api

import (
	"net/http"
	"sort"
	"time"

	"engine/store"
)

// GetTableSizesHandler handles GET /api/connections/{id}/table-sizes
func GetTableSizesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	if store.GetActiveConnection(id) == nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Connection is not active"}, http.StatusBadRequest)
		return
	}

	sizes, updatedAt := store.GetTableSizes(id)
	status := store.GetTableSizeStatus(id)
	sort.Slice(sizes, func(i, j int) bool { return sizes[i].SizeBytes > sizes[j].SizeBytes })

	sendJSONResponse(w, map[string]interface{}{
		"success":    true,
		"sizes":      sizes,
		"updated_at": formatTimeOrEmpty(updatedAt),
		"status":     status.Status,
		"error":      status.Error,
	}, http.StatusOK)
}

// RefreshTableSizesHandler handles POST /api/connections/{id}/table-sizes/refresh
func RefreshTableSizesHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	if store.GetActiveConnection(id) == nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Connection is not active"}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Connection config not found"}, http.StatusInternalServerError)
		return
	}

	runTableSizeEstimator(id, *config)

	sizes, updatedAt := store.GetTableSizes(id)
	status := store.GetTableSizeStatus(id)
	sort.Slice(sizes, func(i, j int) bool { return sizes[i].SizeBytes > sizes[j].SizeBytes })

	sendJSONResponse(w, map[string]interface{}{
		"success":    true,
		"sizes":      sizes,
		"updated_at": formatTimeOrEmpty(updatedAt),
		"status":     status.Status,
		"error":      status.Error,
	}, http.StatusOK)
}

// GetTableSizesStatusHandler handles GET /api/connections/{id}/table-sizes/status
func GetTableSizesStatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	if store.GetActiveConnection(id) == nil {
		sendJSONResponse(w, map[string]interface{}{"success": false, "error": "Connection is not active"}, http.StatusBadRequest)
		return
	}

	status := store.GetTableSizeStatus(id)
	if status.Status == "" {
		if store.GetTableSizeCount(id) > 0 {
			store.SetTableSizeStatus(id, "ready", "")
			status = store.GetTableSizeStatus(id)
		} else if config, err := store.GetConnection(id); err == nil {
			startTableSizeEstimator(id, *config)
			status = store.GetTableSizeStatus(id)
		} else {
			store.SetTableSizeStatus(id, "error", "Connection config not found")
			status = store.GetTableSizeStatus(id)
		}
	}
	sendJSONResponse(w, map[string]interface{}{
		"success":    true,
		"status":     status.Status,
		"count":      store.GetTableSizeCount(id),
		"updated_at": formatTimeOrEmpty(status.UpdatedAt),
		"error":      status.Error,
	}, http.StatusOK)
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
