package db

import (
	"strings"
	"testing"

	"engine/store"
)

func TestBuildDSN_Postgres(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "localhost",
		Port:      5432,
		User:      "myuser",
		Password:  "mypass",
		DBName:    "mydb",
		EnableSSL: false,
	}
	dsn, err := buildDSN(config, "postgres")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "host=localhost") {
		t.Errorf("missing host: %s", dsn)
	}
	if !strings.Contains(dsn, "port=5432") {
		t.Errorf("missing port: %s", dsn)
	}
	if !strings.Contains(dsn, "user=myuser") {
		t.Errorf("missing user: %s", dsn)
	}
	if !strings.Contains(dsn, "password=mypass") {
		t.Errorf("missing password: %s", dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("expected sslmode=disable: %s", dsn)
	}
	if !strings.Contains(dsn, "dbname=mydb") {
		t.Errorf("missing dbname: %s", dsn)
	}
}

func TestBuildDSN_Postgres_SSL(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "db.example.com",
		Port:      5432,
		User:      "u",
		Password:  "p",
		DBName:    "prod",
		EnableSSL: true,
	}
	dsn, err := buildDSN(config, "postgres")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "sslmode=require") {
		t.Errorf("expected sslmode=require: %s", dsn)
	}
}

func TestBuildDSN_Postgres_EmptyDBName(t *testing.T) {
	config := store.ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "u",
		Password: "p",
		DBName:   "",
	}
	dsn, err := buildDSN(config, "postgres")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if strings.Contains(dsn, "dbname=") {
		t.Errorf("should not include dbname when empty: %s", dsn)
	}
}

func TestBuildDSN_MySQL(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "localhost",
		Port:      3306,
		User:      "root",
		Password:  "secret",
		DBName:    "appdb",
		EnableSSL: false,
	}
	dsn, err := buildDSN(config, "mysql")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "root:secret@tcp(localhost:3306)/appdb") {
		t.Errorf("unexpected mysql dsn: %s", dsn)
	}
	if !strings.Contains(dsn, "tls=false") {
		t.Errorf("expected tls=false: %s", dsn)
	}
}

func TestBuildDSN_MySQL_SSL(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "db.example.com",
		Port:      3306,
		User:      "u",
		Password:  "p",
		DBName:    "db",
		EnableSSL: true,
	}
	dsn, err := buildDSN(config, "mysql")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "tls=true") {
		t.Errorf("expected tls=true: %s", dsn)
	}
}

func TestBuildDSN_SqlServer(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "sqlserver.example.com",
		Port:      1433,
		User:      "sa",
		Password:  "P@ssw0rd",
		DBName:    "master",
		EnableSSL: false,
	}
	dsn, err := buildDSN(config, "sqlserver")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "sqlserver://") {
		t.Errorf("expected sqlserver scheme: %s", dsn)
	}
	if !strings.Contains(dsn, "sqlserver.example.com:1433") {
		t.Errorf("expected host:port: %s", dsn)
	}
	if !strings.Contains(dsn, "trustservercertificate=true") {
		t.Errorf("expected trustservercertificate=true: %s", dsn)
	}
	if !strings.Contains(dsn, "database=master") {
		t.Errorf("expected database param: %s", dsn)
	}
}

func TestBuildDSN_SqlServer_SSL(t *testing.T) {
	config := store.ConnectionConfig{
		Host:      "localhost",
		Port:      1433,
		User:      "u",
		Password:  "p",
		DBName:    "db",
		EnableSSL: true,
	}
	dsn, err := buildDSN(config, "sqlserver")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if !strings.Contains(dsn, "encrypt=true") {
		t.Errorf("expected encrypt=true: %s", dsn)
	}
}

func TestBuildDSN_SqlServer_EmptyDBName(t *testing.T) {
	config := store.ConnectionConfig{
		Host:     "localhost",
		Port:     1433,
		User:     "u",
		Password: "p",
		DBName:   "",
	}
	dsn, err := buildDSN(config, "sqlserver")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if strings.Contains(dsn, "database=") {
		t.Errorf("should not include database when empty: %s", dsn)
	}
}

func TestBuildDSN_Unsupported(t *testing.T) {
	config := store.ConnectionConfig{Host: "localhost", Port: 5432, User: "u", Password: "p"}
	_, err := buildDSN(config, "duckdb")
	if err == nil {
		t.Error("expected error for unsupported driver")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildDSN_SQLite_FilePath(t *testing.T) {
	config := store.ConnectionConfig{FilePath: "/tmp/test.db"}
	dsn, err := buildDSN(config, "sqlite")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if dsn != "/tmp/test.db" {
		t.Fatalf("unexpected sqlite dsn: %s", dsn)
	}
}

func TestBuildDSN_SQLite_MissingFilePath(t *testing.T) {
	config := store.ConnectionConfig{}
	_, err := buildDSN(config, "sqlite")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "file_path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDSN_Oracle(t *testing.T) {
	config := store.ConnectionConfig{
		Host:     "oracle.example.com",
		Port:     1521,
		User:     "scott",
		Password: "tiger",
		DBName:   "orclpdb1",
	}
	dsn, err := buildDSN(config, "oracle")
	if err != nil {
		t.Fatalf("buildDSN: %v", err)
	}
	if dsn != "oracle://scott:tiger@oracle.example.com:1521/orclpdb1" {
		t.Fatalf("unexpected oracle dsn: %s", dsn)
	}
}

func TestConnect_SQLite(t *testing.T) {
	conn, err := Connect(store.ConnectionConfig{DBType: "sqlite", FilePath: ":memory:"})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	_ = conn.Close()
}
