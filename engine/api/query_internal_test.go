package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHandleSelectQuery_ErrorsAndEmptyRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Query error -> 200 with error payload.
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1")).WillReturnError(errors.New("bad query"))
	rr := httptest.NewRecorder()
	handleSelectQuery(rr, db, "SELECT 1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// No rows -> rows should be [] (not null) and success true.
	mock.ExpectQuery(regexp.QuoteMeta("SELECT empty")).WillReturnRows(sqlmock.NewRows([]string{"a"}))
	rr = httptest.NewRecorder()
	handleSelectQuery(rr, db, "SELECT empty")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp QueryResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Success || len(resp.Columns) != 1 || resp.Columns[0] != "a" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
	// Field may be omitted due to omitempty, but should never be explicitly null.
	if s := rr.Body.String(); regexp.MustCompile(`"rows"\\s*:\\s*null`).MatchString(s) {
		t.Fatalf("unexpected null rows: %s", s)
	}

	// Row iteration error -> 500.
	rows := sqlmock.NewRows([]string{"a"}).AddRow("x").RowError(0, errors.New("row err"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT rowerr")).WillReturnRows(rows)
	rr = httptest.NewRecorder()
	handleSelectQuery(rr, db, "SELECT rowerr")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestHandleMutationQuery_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE t SET a=1")).WillReturnError(errors.New("exec failed"))
	rr := httptest.NewRecorder()
	handleMutationQuery(rr, db, "UPDATE t SET a=1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
