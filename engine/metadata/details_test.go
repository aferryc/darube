package metadata

import (
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetTableDML_MySQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SHOW CREATE TABLE `app`.`users`")).
		WillReturnRows(sqlmock.NewRows([]string{"Table", "Create Table"}).AddRow("users", "CREATE TABLE users (id int)"))

	dml, err := GetTableDML("mysql", db, "app", "users")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(dml, "CREATE TABLE") || !strings.HasSuffix(dml, ";") {
		t.Fatalf("unexpected dml: %q", dml)
	}
}

func TestGetTableDML_Postgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.columns")).
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type", "character_maximum_length", "is_nullable", "column_default"}).
			AddRow("id", "bigint", nil, "NO", nil).
			AddRow("name", "text", nil, "YES", "'x'"))

	dml, err := GetTableDML("postgres", db, "public", "users")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(dml, "CREATE TABLE public.users") {
		t.Fatalf("unexpected dml: %q", dml)
	}
	if !strings.Contains(dml, "id bigint NOT NULL") {
		t.Fatalf("missing id column: %q", dml)
	}
	if !strings.Contains(dml, "DEFAULT") {
		t.Fatalf("missing default: %q", dml)
	}
}

func TestGetTableDML_Postgres_NoColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.columns")).
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type", "character_maximum_length", "is_nullable", "column_default"}))

	_, err = GetTableDML("postgres", db, "public", "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetTableDML_SQLServerAndUnsupported(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	dml, err := GetTableDML("sqlserver", db, "dbo", "t")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(dml, "not fully implemented") {
		t.Fatalf("unexpected: %q", dml)
	}

	_, err = GetTableDML("sqlite", db, "main", "t")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetTableIndexes_MySQLAndPostgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// MySQL: vary columns; we provide canonical ones.
	mock.ExpectQuery(regexp.QuoteMeta("SHOW INDEX FROM `app`.`users`")).
		WillReturnRows(sqlmock.NewRows([]string{"Table", "Non_unique", "Key_name", "Seq_in_index", "Column_name"}).
			AddRow("users", "0", "PRIMARY", "1", "id").
			AddRow("users", "1", "idx_name", "1", "name"))
	idxs, err := GetTableIndexes("mysql", db, "app", "users")
	if err != nil || len(idxs) != 2 {
		t.Fatalf("mysql: idxs=%v err=%v", idxs, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM\n\t\t\t\tpg_class t,")).
		WillReturnRows(sqlmock.NewRows([]string{"index_name", "column_name", "is_unique", "is_primary"}).
			AddRow("idx_users_id", "id", true, false))
	idxs, err = GetTableIndexes("postgres", db, "public", "users")
	if err != nil || len(idxs) != 1 || idxs[0].Name != "idx_users_id" {
		t.Fatalf("postgres: idxs=%v err=%v", idxs, err)
	}

	// sqlserver placeholder
	idxs, err = GetTableIndexes("sqlserver", db, "dbo", "t")
	if err != nil || len(idxs) != 0 {
		t.Fatalf("sqlserver: idxs=%v err=%v", idxs, err)
	}

	// unknown type currently returns empty slice, nil error
	idxs, err = GetTableIndexes("nope", db, "x", "y")
	if err != nil || idxs != nil {
		t.Fatalf("nope: idxs=%v err=%v", idxs, err)
	}
}
