package metadata

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetDatabases_SQLVariants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT datname FROM pg_database").
		WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("app"))
	out, err := GetDatabases("postgres", db)
	if err != nil || len(out) != 1 || out[0].Name != "app" {
		t.Fatalf("postgres: out=%v err=%v", out, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT schema_name")).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("app"))
	out, err = GetDatabases("mysql", db)
	if err != nil || len(out) != 1 || out[0].Name != "app" {
		t.Fatalf("mysql: out=%v err=%v", out, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT name")).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("app"))
	out, err = GetDatabases("sqlserver", db)
	if err != nil || len(out) != 1 || out[0].Name != "app" {
		t.Fatalf("sqlserver: out=%v err=%v", out, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT SYS_CONTEXT('USERENV','DB_NAME') FROM dual")).
		WillReturnRows(sqlmock.NewRows([]string{"SYS_CONTEXT"}).AddRow("ORCL"))
	out, err = GetDatabases("oracle", db)
	if err != nil || len(out) != 1 || out[0].Name != "ORCL" {
		t.Fatalf("oracle: out=%v err=%v", out, err)
	}

	// SQLite is special-cased and doesn't query.
	out, err = GetDatabases("sqlite", db)
	if err != nil || len(out) != 1 || out[0].Name != "main" {
		t.Fatalf("sqlite: out=%v err=%v", out, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetDatabases_Unsupported(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	_, err = GetDatabases("nope", db)
	if err == nil {
		t.Fatalf("expected error")
	}
}
