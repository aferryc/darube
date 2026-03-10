package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"engine/store"

	"github.com/xuri/excelize/v2"
)

type ExportParams struct {
	TargetType      string          `json:"target_type"`      // "table", "query", or "data"
	Target          string          `json:"target"`           // name of the table or raw sql query (used for table/query)
	Format          string          `json:"format"`           // "csv", "json", "sql", "excel"
	Headers         bool            `json:"headers"`          // true/false for CSV/Excel
	DestinationPath string          `json:"destination_path"` // absolute dir path
	Filename        string          `json:"filename"`         // without extension
	Columns         []string        `json:"columns"`          // used if target_type == "data"
	Data            [][]interface{} `json:"data"`             // used if target_type == "data"
}

type ExportResponse struct {
	Success bool   `json:"success"`
	SavedTo string `json:"saved_to,omitempty"`
	Error   string `json:"error,omitempty"`
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params ExportParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		sendJSONResponse(w, ExportResponse{Success: false, Error: "Invalid JSON body"}, http.StatusBadRequest)
		return
	}

	conn := store.GetActiveConnection(id)
	if conn == nil {
		sendJSONResponse(w, ExportResponse{Success: false, Error: "Connection is not active"}, http.StatusBadRequest)
		return
	}

	ext := "." + params.Format
	if params.Format == "excel" {
		ext = ".xlsx"
	}
	fullPath := filepath.Join(params.DestinationPath, params.Filename+ext)

	var cols []string
	var rowChan chan []interface{}
	var errChan chan error
	
	rowChan = make(chan []interface{})
	errChan = make(chan error, 1)

	if params.TargetType == "data" {
		cols = params.Columns
		go func() {
			for _, row := range params.Data {
				rowChan <- row
			}
			close(rowChan)
			errChan <- nil
		}()
	} else {
		// table or query
		query := params.Target
		if params.TargetType == "table" {
			query = "SELECT * FROM " + params.Target
		}

		rows, err := conn.Query(query)
		if err != nil {
			sendJSONResponse(w, ExportResponse{Success: false, Error: "Query Execution Error: " + err.Error()}, http.StatusInternalServerError)
			return
		}

		cols, err = rows.Columns()
		if err != nil {
			rows.Close()
			sendJSONResponse(w, ExportResponse{Success: false, Error: "Failed to get columns: " + err.Error()}, http.StatusInternalServerError)
			return
		}

		go func(rs *sql.Rows) {
			defer rs.Close()
			for rs.Next() {
				rawResult := make([][]byte, len(cols))
				dest := make([]interface{}, len(cols))
				for i := range rawResult {
					dest[i] = &rawResult[i]
				}
				if err := rs.Scan(dest...); err != nil {
					errChan <- err
					return
				}
				
				// Convert to standard interface{} row
				row := make([]interface{}, len(cols))
				for i, raw := range rawResult {
					if raw == nil {
						row[i] = nil
					} else {
						row[i] = string(raw) // base string cast
					}
				}
				rowChan <- row
			}
			errChan <- nil
		}(rows)
	}

	var saveErr error
	switch params.Format {
	case "csv":
		saveErr = exportCSV(fullPath, params.Headers, cols, rowChan, errChan)
	case "json":
		saveErr = exportJSON(fullPath, cols, rowChan, errChan)
	case "sql":
		tableRef := params.Target
		if params.TargetType != "table" {
			tableRef = "exported_results"
		}
		saveErr = exportSQL(fullPath, tableRef, cols, rowChan, errChan)
	case "excel":
		saveErr = exportExcel(fullPath, params.Headers, cols, rowChan, errChan)
	default:
		saveErr = fmt.Errorf("unsupported format %s", params.Format)
	}

	if saveErr != nil {
		sendJSONResponse(w, ExportResponse{Success: false, Error: saveErr.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, ExportResponse{Success: true, SavedTo: fullPath}, http.StatusOK)
}

func exportCSV(fullPath string, includeHeaders bool, cols []string, rowChan chan []interface{}, errChan chan error) error {
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if includeHeaders {
		if err := writer.Write(cols); err != nil {
			return err
		}
	}

	for row := range rowChan {
		record := make([]string, len(cols))
		for i, val := range row {
			if val == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", val)
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return <-errChan
}

func exportJSON(fullPath string, cols []string, rowChan chan []interface{}, errChan chan error) error {
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write opening bracket
	file.WriteString("[\n")

	first := true
	for row := range rowChan {
		record := make(map[string]interface{})
		for i, val := range row {
			record[cols[i]] = val
		}

		b, err := json.Marshal(record)
		if err != nil {
			return err
		}

		if !first {
			file.WriteString(",\n")
		}
		file.Write(b)
		first = false
	}

	// Write closing bracket
	file.WriteString("\n]")
	return <-errChan
}

func exportSQL(fullPath string, tableName string, cols []string, rowChan chan []interface{}, errChan chan error) error {
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	colStr := strings.Join(cols, ", ")

	for row := range rowChan {
		var values []string
		for _, val := range row {
			if val == nil {
				values = append(values, "NULL")
			} else {
				safeStr := strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "''")
				values = append(values, fmt.Sprintf("'%s'", safeStr))
			}
		}
		
		stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", tableName, colStr, strings.Join(values, ", "))
		file.WriteString(stmt)
	}
	return <-errChan
}

func exportExcel(fullPath string, includeHeaders bool, cols []string, rowChan chan []interface{}, errChan chan error) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := "Sheet1"
	rowIndex := 1

	if includeHeaders {
		for i, col := range cols {
			// Get column name like A, B, C, etc.
			cellName, err := excelize.CoordinatesToCellName(i+1, rowIndex)
			if err == nil {
				f.SetCellValue(sheetName, cellName, col)
			}
		}
		rowIndex++
	}

	for row := range rowChan {
		for i, val := range row {
			cellName, err := excelize.CoordinatesToCellName(i+1, rowIndex)
			if err == nil && val != nil {
				f.SetCellValue(sheetName, cellName, fmt.Sprintf("%v", val))
			}
		}
		rowIndex++
	}

	if checkErr := <-errChan; checkErr != nil {
		return checkErr
	}

	if err := f.SaveAs(fullPath); err != nil {
		return err
	}
	return nil
}
