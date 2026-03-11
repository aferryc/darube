package store

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMemory_SQLConnections(t *testing.T) {
	db1, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db1.Close()
	db2, mock2, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db2.Close()

	AddActiveConnection("c1", db1)
	AddActiveConnection("c1", db2) // should close db1

	if err := db1.Ping(); err == nil {
		t.Fatalf("expected db1 closed")
	}

	mock2.ExpectPing()
	if ok := IsConnected("c1"); !ok {
		t.Fatalf("expected connected")
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}

	mock2.ExpectClose()
	if err := RemoveActiveConnection("c1"); err != nil {
		t.Fatalf("RemoveActiveConnection: %v", err)
	}
	if GetActiveConnection("c1") != nil {
		t.Fatalf("expected removed")
	}
}

func TestMemory_IsConnectedPingFail(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	AddActiveConnection("c2", db)
	mock.ExpectPing().WillReturnError(assertErr{})
	if ok := IsConnected("c2"); ok {
		t.Fatalf("expected false")
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "ping fail" }

func TestMemory_RedisConnections(t *testing.T) {
	AddRedisConnection("r1", 123)
	if !IsRedisConnected("r1") {
		t.Fatalf("expected redis connected")
	}
	if v := GetRedisConnection("r1"); v != 123 {
		t.Fatalf("unexpected value: %#v", v)
	}
	RemoveRedisConnection("r1")
	if IsRedisConnected("r1") {
		t.Fatalf("expected removed")
	}
}
