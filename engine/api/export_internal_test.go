package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExportHelpers_ErrorFromErrChan(t *testing.T) {
	dir := t.TempDir()

	rowChan := make(chan []interface{})
	errChan := make(chan error, 1)
	close(rowChan)
	errChan <- errors.New("boom")

	if err := exportCSV(filepath.Join(dir, "x.csv"), false, []string{"a"}, rowChan, errChan); err == nil {
		t.Fatalf("expected error")
	}

	rowChan = make(chan []interface{})
	errChan = make(chan error, 1)
	close(rowChan)
	errChan <- errors.New("boom")
	if err := exportJSON(filepath.Join(dir, "x.json"), []string{"a"}, rowChan, errChan); err == nil {
		t.Fatalf("expected error")
	}

	rowChan = make(chan []interface{})
	errChan = make(chan error, 1)
	close(rowChan)
	errChan <- errors.New("boom")
	if err := exportSQL(filepath.Join(dir, "x.sql"), "t", []string{"a"}, rowChan, errChan); err == nil {
		t.Fatalf("expected error")
	}

	rowChan = make(chan []interface{})
	errChan = make(chan error, 1)
	close(rowChan)
	errChan <- errors.New("boom")
	if err := exportExcel(filepath.Join(dir, "x.xlsx"), false, []string{"a"}, rowChan, errChan); err == nil {
		t.Fatalf("expected error")
	}
}

func TestExportExcel_WithHeaders_WritesFile(t *testing.T) {
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "ok.xlsx")

	rowChan := make(chan []interface{}, 1)
	errChan := make(chan error, 1)
	rowChan <- []interface{}{"v"}
	close(rowChan)
	errChan <- nil

	if err := exportExcel(fullPath, true, []string{"col"}, rowChan, errChan); err != nil {
		t.Fatalf("exportExcel: %v", err)
	}
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("expected file created: %v", err)
	}
}

