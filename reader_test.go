package exsheetah_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nukokusa/exsheetah"
	"github.com/xuri/excelize/v2"
)

// buildTestXLSX creates a small xlsx file with a title row (skipped), a
// header row, and data rows exercising strings, integers, floats,
// booleans, a built-in date format, and a custom date format. It also adds
// a percent-formatted number to make sure percent formatting is not
// mistaken for a date.
func buildTestXLSX(t *testing.T) string {
	t.Helper()

	f := excelize.NewFile()
	defer f.Close()
	sheet := "item"
	if _, err := f.NewSheet(sheet); err != nil {
		t.Fatal(err)
	}
	if err := f.DeleteSheet("Sheet1"); err != nil {
		t.Fatal(err)
	}

	dateStyle, err := f.NewStyle(&excelize.Style{NumFmt: 14}) // mm-dd-yy
	if err != nil {
		t.Fatal(err)
	}
	customDateStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("yyyy-mm-dd hh:mm:ss")})
	if err != nil {
		t.Fatal(err)
	}
	percentStyle, err := f.NewStyle(&excelize.Style{NumFmt: 10}) // 0.00%
	if err != nil {
		t.Fatal(err)
	}

	// Row 1: a decorative title row that must be skipped when locating
	// the header.
	must(t, f.SetCellValue(sheet, "A1", "item master data"))

	// Row 2: header row.
	headers := []string{"id", "name", "price", "in_stock", "released_at", "updated_at", "discount"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		must(t, f.SetCellValue(sheet, cell, h))
	}

	// Row 3: data.
	must(t, f.SetCellValue(sheet, "A3", 1))
	must(t, f.SetCellValue(sheet, "B3", "Potion"))
	must(t, f.SetCellValue(sheet, "C3", 12.5))
	must(t, f.SetCellValue(sheet, "D3", true))
	must(t, f.SetCellValue(sheet, "E3", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)))
	must(t, f.SetCellStyle(sheet, "E3", "E3", dateStyle))
	must(t, f.SetCellValue(sheet, "F3", time.Date(2024, 3, 4, 15, 16, 17, 0, time.UTC)))
	must(t, f.SetCellStyle(sheet, "F3", "F3", customDateStyle))
	must(t, f.SetCellValue(sheet, "G3", 0.3))
	must(t, f.SetCellStyle(sheet, "G3", "G3", percentStyle))

	// Row 4: another data row, with a false boolean and a plain integer
	// price to make sure formatFloat still collapses to an integer.
	must(t, f.SetCellValue(sheet, "A4", 2))
	must(t, f.SetCellValue(sheet, "B4", "Hi-Potion"))
	must(t, f.SetCellValue(sheet, "C4", 30))
	must(t, f.SetCellValue(sheet, "D4", false))
	must(t, f.SetCellValue(sheet, "E4", time.Date(2024, 5, 6, 0, 0, 0, 0, time.UTC)))
	must(t, f.SetCellStyle(sheet, "E4", "E4", dateStyle))
	must(t, f.SetCellValue(sheet, "F4", time.Date(2024, 7, 8, 9, 10, 11, 0, time.UTC)))
	must(t, f.SetCellStyle(sheet, "F4", "F4", customDateStyle))
	must(t, f.SetCellValue(sheet, "G4", 0.5))
	must(t, f.SetCellStyle(sheet, "G4", "G4", percentStyle))

	path := filepath.Join(t.TempDir(), "test.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func strPtr(s string) *string { return &s }

func TestReader_ReadSheets(t *testing.T) {
	t.Parallel()

	path := buildTestXLSX(t)

	config := &exsheetah.SheetConfig{
		Name: "item",
		Columns: []*exsheetah.ColumnConfig{
			{Name: "id", Type: exsheetah.ColumnTypeNumber},
			{Name: "name", Type: exsheetah.ColumnTypeString},
			{Name: "price", Type: exsheetah.ColumnTypeNumber},
			{Name: "in_stock", Type: exsheetah.ColumnTypeBool},
			{Name: "released_at", Type: exsheetah.ColumnTypeTimestamp},
			{Name: "updated_at", Type: exsheetah.ColumnTypeTimestamp},
			{Name: "discount", Type: exsheetah.ColumnTypeNumber},
		},
	}

	r, err := exsheetah.NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	sheets, err := r.ReadSheets([]*exsheetah.SheetConfig{config}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	sheet := sheets[0]
	if len(sheet.Rows) != 2 {
		t.Fatalf("expected 2 data rows (title row must be skipped), got %d", len(sheet.Rows))
	}

	row0 := rowMap(sheet, 0)
	if got := row0["id"]; got != int64(1) {
		t.Errorf("id = %v, want 1", got)
	}
	if got := row0["name"]; got != "Potion" {
		t.Errorf("name = %v, want Potion", got)
	}
	if got := row0["price"]; got != float64(12.5) {
		t.Errorf("price = %v, want 12.5", got)
	}
	if got := row0["in_stock"]; got != true {
		t.Errorf("in_stock = %v, want true", got)
	}
	wantReleasedAt := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if got, ok := row0["released_at"].(time.Time); !ok || !got.Equal(wantReleasedAt) {
		t.Errorf("released_at = %v, want %v", row0["released_at"], wantReleasedAt)
	}
	wantUpdatedAt := time.Date(2024, 3, 4, 15, 16, 17, 0, time.UTC)
	if got, ok := row0["updated_at"].(time.Time); !ok || !got.Equal(wantUpdatedAt) {
		t.Errorf("updated_at = %v, want %v", row0["updated_at"], wantUpdatedAt)
	}
	// A percent-formatted number must be read as a plain number, not
	// mistaken for a date.
	if got := row0["discount"]; got != float64(0.3) {
		t.Errorf("discount = %v, want 0.3", got)
	}

	row1 := rowMap(sheet, 1)
	if got := row1["id"]; got != int64(2) {
		t.Errorf("id = %v, want 2", got)
	}
	if got := row1["price"]; got != int64(30) {
		t.Errorf("price = %v, want 30 (integer)", got)
	}
	if got := row1["in_stock"]; got != false {
		t.Errorf("in_stock = %v, want false", got)
	}
}

func rowMap(s *exsheetah.Sheet, i int) map[string]any {
	columnTypeMap := map[string]exsheetah.ColumnType{}
	for _, c := range s.Config.Columns {
		columnTypeMap[c.Name] = c.Type
	}
	result := map[string]any{}
	for j, cell := range s.Rows[i] {
		name := s.Columns[j]
		typ, ok := columnTypeMap[name]
		if !ok {
			continue
		}
		result[name] = cell.Value(typ)
	}
	return result
}

func TestReader_SheetNotFound(t *testing.T) {
	t.Parallel()

	path := buildTestXLSX(t)
	r, err := exsheetah.NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, err = r.ReadSheets([]*exsheetah.SheetConfig{{
		Name:    "does-not-exist",
		Columns: []*exsheetah.ColumnConfig{{Name: "id", Type: exsheetah.ColumnTypeNumber}},
	}}, time.UTC)
	if err == nil {
		t.Fatal("expected an error for a missing sheet")
	}
}

func TestReader_Range(t *testing.T) {
	t.Parallel()

	path := buildTestXLSX(t)
	r, err := exsheetah.NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Restrict the range to columns A-D so that only id/name/price/in_stock
	// are visible; the header detection should still find row 2 within
	// the restricted range.
	config := &exsheetah.SheetConfig{
		Name:  "item",
		Range: "A2:D4",
		Columns: []*exsheetah.ColumnConfig{
			{Name: "id", Type: exsheetah.ColumnTypeNumber},
			{Name: "name", Type: exsheetah.ColumnTypeString},
			{Name: "price", Type: exsheetah.ColumnTypeNumber},
			{Name: "in_stock", Type: exsheetah.ColumnTypeBool},
		},
	}
	sheets, err := r.ReadSheets([]*exsheetah.SheetConfig{config}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets[0].Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d (%v)", len(sheets[0].Columns), sheets[0].Columns)
	}
}
