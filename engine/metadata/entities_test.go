package metadata

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetEntities_Postgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.tables")).
		WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name", "table_type", "column_name", "data_type"}).
			AddRow("public", "users", "BASE TABLE", "id", "bigint").
			AddRow("public", "users", "BASE TABLE", "name", "text"))

	mock.ExpectQuery("SELECT schemaname, tablename, indexname FROM pg_indexes WHERE schemaname NOT IN").
		WillReturnRows(sqlmock.NewRows([]string{"schemaname", "tablename", "indexname"}).
			AddRow("public", "users", "idx_users_id"))

	out, err := GetEntities("postgres", db)
	if err != nil || len(out) != 1 {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if out[0].Name != "public" || len(out[0].Tables) != 1 {
		t.Fatalf("unexpected: %#v", out)
	}
	if len(out[0].Tables[0].Columns) != 2 {
		t.Fatalf("unexpected columns: %#v", out[0].Tables[0].Columns)
	}
}

func TestGetEntities_MySQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.tables")).
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_SCHEMA", "TABLE_NAME", "TABLE_TYPE", "COLUMN_NAME", "COLUMN_TYPE"}).
			AddRow("app", "users", "BASE TABLE", "id", "bigint"))

	out, err := GetEntities("mysql", db)
	if err != nil || len(out) != 1 || out[0].Name != "app" {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestGetEntities_SQLServer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM sys.objects t")).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name", "table_name", "table_type", "column_name", "data_type"}).
			AddRow("dbo", "users", "USER_TABLE", "id", "int"))

	out, err := GetEntities("sqlserver", db)
	if err != nil || len(out) != 1 || out[0].Name != "dbo" {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestGetEntities_Unsupported(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	_, err = GetEntities("sqlite", db)
	if err == nil {
		t.Fatalf("expected error")
	}
}
