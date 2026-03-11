package metadata

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetSchemas_Variants(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT schema_name")).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
	s, err := GetSchemas("postgres", db)
	if err != nil || len(s) != 1 || s[0].Name != "public" {
		t.Fatalf("postgres: %#v err=%v", s, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATABASE() as schema_name;")).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("app"))
	s, err = GetSchemas("mysql", db)
	if err != nil || len(s) != 1 || s[0].Name != "app" {
		t.Fatalf("mysql: %#v err=%v", s, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT name AS schema_name")).
		WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("dbo"))
	s, err = GetSchemas("sqlserver", db)
	if err != nil || len(s) != 1 || s[0].Name != "dbo" {
		t.Fatalf("sqlserver: %#v err=%v", s, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM dual")).
		WillReturnRows(sqlmock.NewRows([]string{"schema"}).AddRow("APP"))
	s, err = GetSchemas("oracle", db)
	if err != nil || len(s) != 1 || s[0].Name != "APP" {
		t.Fatalf("oracle: %#v err=%v", s, err)
	}

	s, err = GetSchemas("sqlite", db)
	if err != nil || len(s) != 1 || s[0].Name != "main" {
		t.Fatalf("sqlite: %#v err=%v", s, err)
	}
}

func TestGetTablesList_PostgresAndMySQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// postgres query is fmt.Sprintf with schema; match key parts.
	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.tables")).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "table_type"}).
			AddRow("users", "BASE TABLE").
			AddRow("v_users", "VIEW"))
	out, err := GetTablesList("postgres", db, "public")
	if err != nil || len(out) != 2 {
		t.Fatalf("postgres: out=%v err=%v", out, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.tables")).
		WithArgs("app").
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME", "TABLE_TYPE"}).
			AddRow("users", "BASE TABLE"))
	out, err = GetTablesList("mysql", db, "app")
	if err != nil || len(out) != 1 || out[0].Name != "users" {
		t.Fatalf("mysql: out=%v err=%v", out, err)
	}
}

func TestGetTablesList_Oracle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM user_tables")).
		WillReturnRows(sqlmock.NewRows([]string{"table_name", "kind"}).
			AddRow("USERS", "TABLE").
			AddRow("V_USERS", "VIEW"))
	out, err := GetTablesList("oracle", db, "APP")
	if err != nil || len(out) != 2 {
		t.Fatalf("oracle: out=%v err=%v", out, err)
	}
	if out[0].Type != "table" || out[1].Type != "view" {
		t.Fatalf("oracle type mapping: %#v", out)
	}
}

func TestGetColumnsList_PostgresMySQLOracle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.columns")).
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type"}).
			AddRow("id", "int8").
			AddRow("name", "text"))
	cols, err := GetColumnsList("postgres", db, "public", "users")
	if err != nil || len(cols) != 2 || cols[0].Name != "id" {
		t.Fatalf("postgres: cols=%v err=%v", cols, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM information_schema.columns")).
		WithArgs("app", "users").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}).
			AddRow("id", "bigint"))
	cols, err = GetColumnsList("mysql", db, "app", "users")
	if err != nil || len(cols) != 1 || cols[0].Type != "bigint" {
		t.Fatalf("mysql: cols=%v err=%v", cols, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("FROM user_tab_columns")).
		WithArgs("USERS").
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type"}).
			AddRow("ID", "NUMBER"))
	cols, err = GetColumnsList("oracle", db, "APP", "users")
	if err != nil || len(cols) != 1 || cols[0].Name != "ID" {
		t.Fatalf("oracle: cols=%v err=%v", cols, err)
	}
}
