package metadata

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteMetadata_Basics(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT
);
CREATE VIEW v_users AS SELECT id, name FROM users;
`)
	if err != nil {
		t.Fatalf("setup schema: %v", err)
	}

	dbs, err := GetDatabases("sqlite", db)
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 1 || dbs[0].Name != "main" {
		t.Fatalf("unexpected databases: %#v", dbs)
	}

	schemas, err := GetSchemas("sqlite", db)
	if err != nil {
		t.Fatalf("GetSchemas: %v", err)
	}
	if len(schemas) != 1 || schemas[0].Name != "main" {
		t.Fatalf("unexpected schemas: %#v", schemas)
	}

	tables, err := GetTablesList("sqlite", db, "main")
	if err != nil {
		t.Fatalf("GetTablesList: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 entities, got %d: %#v", len(tables), tables)
	}

	var sawUsers, sawView bool
	for _, e := range tables {
		switch e.Name {
		case "users":
			sawUsers = true
			if e.Type != "table" {
				t.Fatalf("expected users to be table, got %q", e.Type)
			}
		case "v_users":
			sawView = true
			if e.Type != "view" {
				t.Fatalf("expected v_users to be view, got %q", e.Type)
			}
		}
	}
	if !sawUsers || !sawView {
		t.Fatalf("missing expected entities, got: %#v", tables)
	}

	cols, err := GetColumnsList("sqlite", db, "main", "users")
	if err != nil {
		t.Fatalf("GetColumnsList: %v", err)
	}
	if len(cols) < 2 {
		t.Fatalf("expected at least 2 columns, got %d: %#v", len(cols), cols)
	}
	if cols[0].Name != "id" {
		t.Fatalf("expected first column id, got %q", cols[0].Name)
	}
	if cols[1].Name != "name" {
		t.Fatalf("expected second column name, got %q", cols[1].Name)
	}
}
