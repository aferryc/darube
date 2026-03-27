package store

import "testing"

func TestTableSizesStore_SetGetAndClear(t *testing.T) {
	connID := "conn-table-sizes-store"

	SetTableSizes(connID, []TableSize{
		{Schema: "public", Table: "users", SizeBytes: 100},
		{Schema: "public", Table: "orders", SizeBytes: 200},
	})

	if _, ok := GetTableSize(connID, "public", "users"); !ok {
		t.Fatalf("expected table size to be present")
	}

	sizes, updatedAt := GetTableSizes(connID)
	if len(sizes) != 2 {
		t.Fatalf("expected 2 sizes, got %d", len(sizes))
	}
	if updatedAt.IsZero() {
		t.Fatalf("expected updatedAt to be set")
	}

	ClearTableSizes(connID)
	if _, ok := GetTableSize(connID, "public", "users"); ok {
		t.Fatalf("expected table size to be cleared")
	}
}
