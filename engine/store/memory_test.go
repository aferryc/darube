package store

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}

func TestAddActiveConnection_GetActiveConnection(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	AddActiveConnection("conn-1", db)
	got := GetActiveConnection("conn-1")
	if got != db {
		t.Error("GetActiveConnection returned wrong db")
	}
	if GetActiveConnection("nonexistent") != nil {
		t.Error("expected nil for nonexistent id")
	}
}

func TestAddActiveConnection_ReplaceExisting(t *testing.T) {
	db1 := openTestDB(t)
	defer db1.Close()
	db2 := openTestDB(t)
	defer db2.Close()

	AddActiveConnection("conn-1", db1)
	AddActiveConnection("conn-1", db2)

	got := GetActiveConnection("conn-1")
	if got != db2 {
		t.Error("expected db2 after replace")
	}
}

func TestRemoveActiveConnection(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	AddActiveConnection("conn-1", db)
	if err := RemoveActiveConnection("conn-1"); err != nil {
		t.Fatalf("RemoveActiveConnection: %v", err)
	}
	if GetActiveConnection("conn-1") != nil {
		t.Error("connection should be removed")
	}
}

func TestRemoveActiveConnection_Nonexistent(t *testing.T) {
	if err := RemoveActiveConnection("nonexistent"); err != nil {
		t.Errorf("RemoveActiveConnection on nonexistent should succeed: %v", err)
	}
}

func TestIsConnected(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	AddActiveConnection("conn-1", db)
	if !IsConnected("conn-1") {
		t.Error("IsConnected should be true for active connection")
	}
	if IsConnected("nonexistent") {
		t.Error("IsConnected should be false for nonexistent")
	}
}

func TestIsConnected_AfterRemove(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	AddActiveConnection("conn-1", db)
	RemoveActiveConnection("conn-1")
	if IsConnected("conn-1") {
		t.Error("IsConnected should be false after remove")
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			db := openTestDB(t)
			defer db.Close()
			id := fmt.Sprintf("conn-concurrent-%d", n)
			AddActiveConnection(id, db)
			_ = GetActiveConnection(id)
			_ = IsConnected(id)
		}(i)
	}
	wg.Wait()
}
