package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"engine/store"
)

type EstimateResponse struct {
	Success           bool   `json:"success"`
	Available         bool   `json:"available"`
	EstimatedRows     int64  `json:"estimated_rows,omitempty"`
	EstimatedRowBytes int64  `json:"estimated_row_bytes,omitempty"`
	EstimatedBytes    int64  `json:"estimated_bytes,omitempty"`
	Reason            string `json:"reason,omitempty"`
	Error             string `json:"error,omitempty"`
}

// EstimateHandler handles POST /api/connections/{id}/estimate
func EstimateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "id path parameter is required"}, http.StatusBadRequest)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	trimmedQuery := strings.TrimSpace(req.Query)
	if trimmedQuery == "" {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Query cannot be empty"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Connection is not active"}, http.StatusBadRequest)
		return
	}

	config, err := store.GetConnection(id)
	if err != nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Connection config not found"}, http.StatusInternalServerError)
		return
	}

	explainQuery, ok := buildEstimateExplainQuery(config.DBType, trimmedQuery)
	if sizeBytes, ok := estimateFromTableSize(id, config, trimmedQuery); ok {
		sendJSONResponse(w, EstimateResponse{
			Success:        true,
			Available:      true,
			EstimatedBytes: sizeBytes,
		}, http.StatusOK)
		return
	}
	if !ok {
		sendJSONResponse(w, EstimateResponse{Success: true, Available: false, Reason: "unsupported_database"}, http.StatusOK)
		return
	}

	rows, err := conn.Query(explainQuery)
	if err != nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: err.Error()}, http.StatusOK)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		sendJSONResponse(w, EstimateResponse{Success: true, Available: false, Reason: "no_plan_returned"}, http.StatusOK)
		return
	}

	var planJSON string
	if err := rows.Scan(&planJSON); err != nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Failed to read plan: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	var rawPlan interface{}
	if err := json.Unmarshal([]byte(planJSON), &rawPlan); err != nil {
		sendJSONResponse(w, EstimateResponse{Success: false, Error: "Failed to parse underlying JSON plan: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	switch config.DBType {
	case "postgres", "postgresql":
		rowsEst, widthEst, ok := extractPostgresEstimate(rawPlan)
		if !ok || rowsEst <= 0 || widthEst <= 0 {
			sendJSONResponse(w, EstimateResponse{Success: true, Available: false, Reason: "missing_plan_metrics"}, http.StatusOK)
			return
		}
		estimatedBytes := rowsEst * widthEst
		sendJSONResponse(w, EstimateResponse{
			Success:           true,
			Available:         true,
			EstimatedRows:     rowsEst,
			EstimatedRowBytes: widthEst,
			EstimatedBytes:    estimatedBytes,
		}, http.StatusOK)
		return
	case "mysql", "mariadb":
		bytesEst, ok := extractMySQLDataRead(rawPlan)
		if !ok || bytesEst <= 0 {
			sendJSONResponse(w, EstimateResponse{Success: true, Available: false, Reason: "missing_plan_metrics"}, http.StatusOK)
			return
		}
		sendJSONResponse(w, EstimateResponse{
			Success:        true,
			Available:      true,
			EstimatedBytes: bytesEst,
		}, http.StatusOK)
		return
	default:
		sendJSONResponse(w, EstimateResponse{Success: true, Available: false, Reason: "unsupported_database"}, http.StatusOK)
		return
	}
}

func estimateFromTableSize(connID string, config *store.ConnectionConfig, query string) (int64, bool) {
	if !isSimpleSelectAll(query) {
		return 0, false
	}

	schema, table, ok := extractFromTable(query)
	if !ok || table == "" {
		return 0, false
	}

	schema = sanitizeIdent(schema)
	table = sanitizeIdent(table)

	if schema == "" {
		switch config.DBType {
		case "mysql", "mariadb":
			schema = sanitizeIdent(config.DBName)
		case "postgres", "postgresql":
			schema = sanitizeIdent(store.GetDefaultSchema(connID))
		}
	}
	if schema == "" {
		return 0, false
	}

	size, ok := store.GetTableSize(connID, schema, table)
	if !ok || size.SizeBytes <= 0 {
		return 0, false
	}
	return size.SizeBytes, true
}

func isSimpleSelectAll(query string) bool {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return false
	}
	selectAllRe := regexp.MustCompile(`(?i)^select\s+\*\s+from\s+`)
	if !selectAllRe.MatchString(trimmed) {
		return false
	}
	blockers := []string{" join ", " where ", " group by ", " union ", " limit ", " offset ", " having "}
	lower := strings.ToLower(trimmed)
	for _, blocker := range blockers {
		if strings.Contains(lower, blocker) {
			return false
		}
	}
	return true
}

func extractFromTable(query string) (string, string, bool) {
	re := regexp.MustCompile(`(?i)\bfrom\s+([a-zA-Z0-9_\.\"` + "`" + `\[\]]+)`)
	match := re.FindStringSubmatch(query)
	if len(match) < 2 {
		return "", "", false
	}
	raw := match[1]
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}
	if strings.Contains(raw, ".") {
		parts := strings.SplitN(raw, ".", 2)
		return parts[0], parts[1], true
	}
	return "", raw, true
}

func sanitizeIdent(ident string) string {
	ident = strings.TrimSpace(ident)
	ident = strings.Trim(ident, "`")
	ident = strings.Trim(ident, "\"")
	ident = strings.TrimPrefix(ident, "[")
	ident = strings.TrimSuffix(ident, "]")
	return ident
}

func buildEstimateExplainQuery(dbType, trimmedQuery string) (string, bool) {
	switch dbType {
	case "postgres", "postgresql":
		return "EXPLAIN (FORMAT JSON) " + trimmedQuery, true
	case "mysql", "mariadb":
		return "EXPLAIN FORMAT=JSON " + trimmedQuery, true
	default:
		return "", false
	}
}

func extractPostgresEstimate(rawPlan interface{}) (int64, int64, bool) {
	arr, ok := rawPlan.([]interface{})
	if !ok || len(arr) == 0 {
		return 0, 0, false
	}
	first, ok := arr[0].(map[string]interface{})
	if !ok {
		return 0, 0, false
	}
	plan, ok := first["Plan"].(map[string]interface{})
	if !ok {
		return 0, 0, false
	}
	rowsF, ok := plan["Plan Rows"].(float64)
	if !ok {
		return 0, 0, false
	}
	widthF, ok := plan["Plan Width"].(float64)
	if !ok {
		return 0, 0, false
	}
	return int64(rowsF), int64(widthF), true
}

func extractMySQLDataRead(rawPlan interface{}) (int64, bool) {
	maxBytes := int64(0)

	var walk func(v interface{})
	walk = func(v interface{}) {
		switch t := v.(type) {
		case map[string]interface{}:
			for k, val := range t {
				if k == "data_read_per_join" {
					if b, ok := parseMySQLBytes(val); ok && b > maxBytes {
						maxBytes = b
					}
				}
				walk(val)
			}
		case []interface{}:
			for _, item := range t {
				walk(item)
			}
		}
	}

	walk(rawPlan)

	if maxBytes <= 0 {
		return 0, false
	}
	return maxBytes, true
}

func parseMySQLBytes(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case float64:
		if t <= 0 {
			return 0, false
		}
		return int64(t), true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		s = strings.ToUpper(s)
		unit := s[len(s)-1:]
		multiplier := float64(1)
		if unit == "K" || unit == "M" || unit == "G" || unit == "T" {
			switch unit {
			case "K":
				multiplier = 1024
			case "M":
				multiplier = 1024 * 1024
			case "G":
				multiplier = 1024 * 1024 * 1024
			case "T":
				multiplier = 1024 * 1024 * 1024 * 1024
			}
			s = s[:len(s)-1]
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil || val <= 0 {
			return 0, false
		}
		return int64(val * multiplier), true
	default:
		return 0, false
	}
}
