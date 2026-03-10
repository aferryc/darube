package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"engine/store"
)

type RedisExportParams struct {
	Format          string      `json:"format"`           // "csv", "json", "excel"
	Headers         bool        `json:"headers"`          // true/false for CSV/Excel
	DestinationPath string      `json:"destination_path"` // absolute dir path
	Filename        string      `json:"filename"`         // without extension
	DataType        string      `json:"data_type"`        // "hash", "list", "string", "integer", "nil", ...
	Value           interface{} `json:"value"`            // result value (already JSON-safe)
	Command         string      `json:"command,omitempty"`// original command (optional)
}

// RedisExportHandler handles POST /api/redis/{id}/export
func RedisExportHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Require an active redis connection so exports match the selected connection lifecycle.
	if store.GetRedisConnection(id) == nil {
		sendJSONResponse(w, ExportResponse{Success: false, Error: "Redis connection is not active"}, http.StatusBadRequest)
		return
	}

	var params RedisExportParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		sendJSONResponse(w, ExportResponse{Success: false, Error: "Invalid JSON body"}, http.StatusBadRequest)
		return
	}

	if params.DestinationPath == "" || params.Filename == "" {
		sendJSONResponse(w, ExportResponse{Success: false, Error: "destination_path and filename are required"}, http.StatusBadRequest)
		return
	}

	ext := "." + params.Format
	if params.Format == "excel" {
		ext = ".xlsx"
	}
	fullPath := filepath.Join(params.DestinationPath, params.Filename+ext)

	switch params.Format {
	case "json":
		payload := map[string]interface{}{
			"command":   params.Command,
			"data_type": params.DataType,
			"value":     params.Value,
		}
		b, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			sendJSONResponse(w, ExportResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(fullPath, b, 0644); err != nil {
			sendJSONResponse(w, ExportResponse{Success: false, Error: err.Error()}, http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, ExportResponse{Success: true, SavedTo: fullPath}, http.StatusOK)
		return

	case "csv", "excel":
		cols, rows := redisToTabular(params.DataType, params.Value)
		rowChan := make(chan []interface{})
		errChan := make(chan error, 1)
		go func() {
			for _, r := range rows {
				rowChan <- r
			}
			close(rowChan)
			errChan <- nil
		}()

		var saveErr error
		if params.Format == "csv" {
			saveErr = exportCSV(fullPath, params.Headers, cols, rowChan, errChan)
		} else {
			saveErr = exportExcel(fullPath, params.Headers, cols, rowChan, errChan)
		}
		if saveErr != nil {
			sendJSONResponse(w, ExportResponse{Success: false, Error: saveErr.Error()}, http.StatusInternalServerError)
			return
		}
		sendJSONResponse(w, ExportResponse{Success: true, SavedTo: fullPath}, http.StatusOK)
		return

	default:
		sendJSONResponse(w, ExportResponse{Success: false, Error: fmt.Sprintf("unsupported format %s", params.Format)}, http.StatusBadRequest)
		return
	}
}

func redisToTabular(dataType string, value interface{}) ([]string, [][]interface{}) {
	switch dataType {
	case "hash":
		if m, ok := value.(map[string]interface{}); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			rows := make([][]interface{}, 0, len(keys))
			for _, k := range keys {
				rows = append(rows, []interface{}{k, redisCellString(m[k])})
			}
			return []string{"Field", "Value"}, rows
		}
	case "list":
		if arr, ok := value.([]interface{}); ok {
			rows := make([][]interface{}, 0, len(arr))
			for i, v := range arr {
				rows = append(rows, []interface{}{i, redisCellString(v)})
			}
			return []string{"Index", "Value"}, rows
		}
	}

	// Scalar fallback
	return []string{"Value"}, [][]interface{}{{redisCellString(value)}}
}

func redisCellString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		// Preserve structured values as JSON strings when possible.
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return fmt.Sprint(t)
	}
}

