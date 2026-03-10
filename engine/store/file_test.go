package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConnections_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	conns, err := ReadConnections()
	if err != nil {
		t.Fatalf("ReadConnections: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected empty, got %d connections", len(conns))
	}
}

func TestReadConnections_NotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	conns, err := ReadConnections()
	if err != nil {
		t.Fatalf("ReadConnections: %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected empty, got %d connections", len(conns))
	}
}

func TestReadConnections_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	data := `[
		{"id":"c1","connection_name":"Test","db_type":"postgres","host":"localhost","port":5432,"dbname":"test","user":"u","password":"p"}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	conns, err := ReadConnections()
	if err != nil {
		t.Fatalf("ReadConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].ID != "c1" || conns[0].ConnectionName != "Test" {
		t.Errorf("unexpected connection: %+v", conns[0])
	}
}

func TestReadConnections_Corrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadConnections()
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestGetConnection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	data := `[
		{"id":"c1","connection_name":"First","db_type":"postgres","host":"h1","port":5432,"dbname":"d1","user":"u1","password":"p1"},
		{"id":"c2","connection_name":"Second","db_type":"mysql","host":"h2","port":3306,"dbname":"d2","user":"u2","password":"p2"}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := GetConnection("c2")
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if c.ID != "c2" || c.ConnectionName != "Second" {
		t.Errorf("unexpected connection: %+v", c)
	}

	_, err = GetConnection("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestWriteConnection_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	config := ConnectionConfig{
		ID:             "new1",
		ConnectionName: "New Conn",
		DBType:         "postgres",
		Host:           "localhost",
		Port:           5432,
		DBName:         "db",
		User:           "u",
		Password:       "p",
	}
	if err := WriteConnection(config); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	conns, err := ReadConnections()
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 1 || conns[0].ID != "new1" {
		t.Errorf("expected 1 connection new1, got %+v", conns)
	}
}

func TestWriteConnection_Update(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	data := `[{"id":"c1","connection_name":"Old","db_type":"postgres","host":"h","port":5432,"dbname":"d","user":"u","password":"p"}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	config := ConnectionConfig{
		ID:             "c1",
		ConnectionName: "Updated",
		DBType:         "postgres",
		Host:           "h",
		Port:           5432,
		DBName:         "d",
		User:           "u",
		Password:       "p",
	}
	if err := WriteConnection(config); err != nil {
		t.Fatalf("WriteConnection: %v", err)
	}

	conns, err := ReadConnections()
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 1 || conns[0].ConnectionName != "Updated" {
		t.Errorf("expected updated name, got %+v", conns)
	}
}

func TestDeleteConnection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	data := `[
		{"id":"c1","connection_name":"First","db_type":"postgres","host":"h1","port":5432,"dbname":"d1","user":"u1","password":"p1"},
		{"id":"c2","connection_name":"Second","db_type":"mysql","host":"h2","port":3306,"dbname":"d2","user":"u2","password":"p2"}
	]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteConnection("c1"); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}

	conns, err := ReadConnections()
	if err != nil {
		t.Fatal(err)
	}
	if len(conns) != 1 || conns[0].ID != "c2" {
		t.Errorf("expected c2 only, got %+v", conns)
	}
}

func TestDeleteConnection_NotExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")
	restore := SetConnectionsFileForTest(path)
	defer restore()

	if err := DeleteConnection("nonexistent"); err != nil {
		t.Errorf("DeleteConnection on empty file should succeed: %v", err)
	}
}
