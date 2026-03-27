package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"engine/store"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEstimateHandler_UsesTableSizeCache(t *testing.T) {
	connID := "conn-estimate-cache"

	restore := store.SetConnectionsFileForTest(filepath.Join(t.TempDir(), "connections.json"))
	t.Cleanup(restore)

	if err := store.WriteConnection(store.ConnectionConfig{
		ID:             connID,
		ConnectionName: "Test PG",
		DBType:         "postgres",
		DBName:         "app",
		User:           "user",
	}); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	store.AddActiveConnection(connID, db)
	t.Cleanup(func() { _ = store.RemoveActiveConnection(connID) })

	store.SetTableSizes(connID, []store.TableSize{
		{Schema: "public", Table: "users", SizeBytes: 4096},
	})
	store.SetDefaultSchema(connID, "public")

	body, _ := json.Marshal(map[string]string{"query": "SELECT * FROM users"})
	req := httptest.NewRequest(http.MethodPost, "/api/connections/"+connID+"/estimate", bytes.NewReader(body))
	req.SetPathValue("id", connID)
	rr := httptest.NewRecorder()

	EstimateHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Success        bool   `json:"success"`
		Available      bool   `json:"available"`
		EstimatedBytes int64  `json:"estimated_bytes"`
		Error          string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !resp.Success || !resp.Available {
		t.Fatalf("expected available success, got success=%v available=%v error=%s", resp.Success, resp.Available, resp.Error)
	}
	if resp.EstimatedBytes != 4096 {
		t.Fatalf("expected estimated_bytes=4096, got %d", resp.EstimatedBytes)
	}
}
