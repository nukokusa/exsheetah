package exsheetah

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/xuri/nfp"
)

// Reader reads sheets out of a local xlsx file.
type Reader struct {
	file     *excelize.File
	date1904 bool
}

// NewReader opens the xlsx file at path.
func NewReader(path string) (*Reader, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx file: %w", err)
	}

	props, err := f.GetWorkbookProps()
	if err != nil {
		return nil, fmt.Errorf("failed to read workbook properties: %w", err)
	}

	return &Reader{
		file:     f,
		date1904: props.Date1904 != nil && *props.Date1904,
	}, nil
}

// Close releases resources held by the underlying xlsx file.
func (r *Reader) Close() error {
	return r.file.Close()
}

// ReadSheets reads every sheet described by configs and builds Sheet values
// from them. loc is used to interpret date/time cell values, which xlsx
// stores as plain serial numbers with no timezone of their own.
func (r *Reader) ReadSheets(configs []*SheetConfig, loc *time.Location) ([]*Sheet, error) {
	sheetSet := make(map[string]bool)
	for _, sheet := range r.file.GetSheetList() {
		sheetSet[sheet] = true
	}

	var sheets []*Sheet
	for _, config := range configs {
		sheetName := config.SheetName()
		if !sheetSet[sheetName] {
			return nil, fmt.Errorf("sheet not found: %s", sheetName)
		}

		grid, err := r.readGrid(sheetName, config.Range, loc)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet %s: %w", config.Name, err)
		}

		sheet, err := NewSheet(config, grid, loc)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sheet)
	}

	return sheets, nil
}

// readGrid reads the given range (or, if empty, the whole used range) of
// sheetName into a grid of Go values (string, float64, bool, or nil per
// cell), mirroring the shape the Google Sheets API returns for sheetah.
func (r *Reader) readGrid(sheetName, rangeStr string, loc *time.Location) ([][]any, error) {
	startCol, startRow, endCol, endRow, err := r.resolveRange(sheetName, rangeStr)
	if err != nil {
		return nil, err
	}

	grid := make([][]any, 0, endRow-startRow+1)
	for row := startRow; row <= endRow; row++ {
		rowValues := make([]any, 0, endCol-startCol+1)
		for col := startCol; col <= endCol; col++ {
			cellName, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				return nil, err
			}
			value, err := r.cellValue(sheetName, cellName, loc)
			if err != nil {
				return nil, err
			}
			rowValues = append(rowValues, value)
		}
		grid = append(grid, rowValues)
	}

	return grid, nil
}

func (r *Reader) resolveRange(sheetName, rangeStr string) (startCol, startRow, endCol, endRow int, err error) {
	if rangeStr == "" {
		return r.usedRange(sheetName)
	}

	parts := strings.SplitN(rangeStr, ":", 2)
	startCol, startRow, err = excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if len(parts) == 2 {
		endCol, endRow, err = excelize.CellNameToCoordinates(parts[1])
		if err != nil {
			return 0, 0, 0, 0, err
		}
	} else {
		endCol, endRow = startCol, startRow
	}

	return startCol, startRow, endCol, endRow, nil
}

// usedRange determines the extent of sheetName by scanning its actual row
// data, since the xlsx file's declared dimension is not always reliable
// (some writers, including excelize itself, don't always keep it in sync
// with the cells that were actually set).
func (r *Reader) usedRange(sheetName string) (startCol, startRow, endCol, endRow int, err error) {
	rows, err := r.file.GetRows(sheetName)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("sheet is empty: %s", sheetName)
	}

	maxCol := 0
	for _, row := range rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}
	if maxCol == 0 {
		return 0, 0, 0, 0, fmt.Errorf("sheet is empty: %s", sheetName)
	}

	return 1, 1, maxCol, len(rows), nil
}

// cellValue returns the Go value held by a single cell: a string, a
// float64, a bool, or nil for an empty cell. Cells styled with a date/time
// number format are converted into an RFC3339 string in loc, so that the
// same string-based timestamp parsing used elsewhere applies unchanged.
func (r *Reader) cellValue(sheetName, cellName string, loc *time.Location) (any, error) {
	cellType, err := r.file.GetCellType(sheetName, cellName)
	if err != nil {
		return nil, err
	}

	switch cellType {
	case excelize.CellTypeBool:
		raw, err := r.file.GetCellValue(sheetName, cellName, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, err
		}
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, nil
		}
		return b, nil
	case excelize.CellTypeSharedString, excelize.CellTypeInlineString, excelize.CellTypeFormula, excelize.CellTypeError:
		return r.file.GetCellValue(sheetName, cellName, excelize.Options{RawCellValue: true})
	case excelize.CellTypeDate:
		// Explicit ISO-8601 typed cells (rare); the raw value is already a
		// date/time string that ParseTimeByString can handle.
		return r.file.GetCellValue(sheetName, cellName, excelize.Options{RawCellValue: true})
	default:
		// CellTypeUnset covers both a truly empty cell and the far more
		// common case of a plain number, since xlsx omits the type
		// attribute for numbers.
		raw, err := r.file.GetCellValue(sheetName, cellName, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, err
		}
		if raw == "" {
			return nil, nil
		}
		num, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			// Not actually numeric (shouldn't normally happen); fall back
			// to treating it as text rather than dropping the value.
			return raw, nil
		}

		isDate, err := r.isDateCell(sheetName, cellName)
		if err != nil {
			return nil, err
		}
		if !isDate {
			return num, nil
		}

		t, err := TimeFromExcelSerial(num, r.date1904, loc)
		if err != nil {
			return nil, err
		}
		return t.Format("2006-01-02T15:04:05Z07:00"), nil
	}
}

// dateBuiltInNumFmtIDs are the built-in xlsx number format IDs (per
// ECMA-376) that render a value as a date and/or time.
var dateBuiltInNumFmtIDs = map[int]bool{
	14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true,
	21: true, 22: true, 27: true, 28: true, 29: true, 30: true, 31: true,
	32: true, 33: true, 34: true, 35: true, 36: true, 45: true, 46: true,
	47: true, 50: true, 57: true,
}

// isDateCell reports whether the cell's number format renders it as a
// date and/or time rather than a plain number.
func (r *Reader) isDateCell(sheetName, cellName string) (bool, error) {
	styleID, err := r.file.GetCellStyle(sheetName, cellName)
	if err != nil {
		return false, err
	}
	if styleID == 0 {
		return false, nil
	}

	style, err := r.file.GetStyle(styleID)
	if err != nil {
		return false, err
	}

	if style.CustomNumFmt != nil && *style.CustomNumFmt != "" {
		return numFmtCodeIsDate(*style.CustomNumFmt), nil
	}
	return dateBuiltInNumFmtIDs[style.NumFmt], nil
}

// numFmtCodeIsDate parses an Excel number format code and reports whether
// any of its tokens represent a date/time component.
func numFmtCodeIsDate(code string) bool {
	parser := nfp.NumberFormatParser()
	for _, section := range parser.Parse(code) {
		for _, token := range section.Items {
			if token.TType == nfp.TokenTypeDateTimes || token.TType == nfp.TokenTypeElapsedDateTimes {
				return true
			}
		}
	}
	return false
}
