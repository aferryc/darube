package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Output format for the Electron app
type CommandOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	ID      string `json:"id,omitempty"`
}

// sendJSONResponse is a helper to write JSON back to the client.
func sendJSONResponse(w http.ResponseWriter, out interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	bytes, err := json.Marshal(out)
	if err != nil {
		// Include the marshal error so issues are diagnosable (e.g. unsupported value types).
		safeErr, _ := json.Marshal(err.Error())
		fmt.Fprintf(w, `{"success":false,"error":"Failed to marshal response: %s"}`+"\n", string(safeErr))
		return
	}
	w.Write(bytes)
}
