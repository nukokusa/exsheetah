package exsheetah

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/samber/lo"
)

type Sheet struct {
	Config  *SheetConfig
	Columns []string
	Rows    [][]Cell
}

// NewSheet builds a Sheet from a raw grid of cell values (as produced by
// Reader.readGrid): one []any per row, each element a string, float64,
// bool, or nil. The first row within the grid that contains one of the
// configured column names is treated as the header row; rows before it are
// skipped and rows after it become data rows.
func NewSheet(config *SheetConfig, grid [][]any, loc *time.Location) (*Sheet, error) {
	formatString := func(cell any) string {
		switch c := cell.(type) {
		case string:
			return c
		case float64:
			return strconv.FormatFloat(c, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(c)
		default:
			return ""
		}
	}

	isColumnRow := func(row []any) bool {
		columnMap := lo.SliceToMap(config.Columns, func(c *ColumnConfig) (string, ColumnType) {
			return c.Name, c.Type
		})
		_, exist := lo.Find(row, func(cell any) bool {
			_, ok := columnMap[formatString(cell)]
			return ok
		})
		return exist
	}

	var columns []string
	rows := [][]Cell{}
	for _, row := range grid {
		if columns == nil {
			if isColumnRow(row) {
				columns = lo.Map(row, func(cell any, _ int) string {
					return formatString(cell)
				})
			}
			continue
		}
		rows = append(rows, lo.Map(row, func(cell any, _ int) Cell {
			switch c := cell.(type) {
			case string:
				return NewStringCell(c, loc)
			case float64:
				return NumberCell(c)
			case bool:
				return BoolCell(c)
			default:
				return NilCell{}
			}
		}))
	}
	if columns == nil {
		return nil, fmt.Errorf("columns not found: %s", config.Name)
	}

	return &Sheet{
		Config:  config,
		Columns: columns,
		Rows:    rows,
	}, nil
}

func (s *Sheet) Name() string {
	return s.Config.Name
}

func (s Sheet) marshal() []map[string]any {
	columnConfigMap := lo.SliceToMap(s.Config.Columns, func(c *ColumnConfig) (string, *ColumnConfig) {
		return c.Name, c
	})

	headers := make(map[int]string)
	for i, column := range s.Columns {
		headers[i] = column
	}

	type rowWithID struct {
		row map[string]any
		id  any
	}

	var rows []rowWithID
	for _, row := range s.Rows {
		rowMap := make(map[string]any)
		idSet := false
		var idValue any
		for i, cell := range row {
			name, ok := headers[i]
			if !ok {
				continue
			}
			cc, ok := columnConfigMap[name]
			if !ok {
				continue
			}
			value := cell.Value(cc.Type)
			if value == nil {
				continue
			}
			if name == s.Config.IDColumn {
				idSet, idValue = true, value
			}
			if cc.Type == ColumnTypeTimestamp && cc.Format != "" {
				if t, ok := value.(time.Time); ok {
					value = t.Format(cc.Format)
				}
			}
			rowMap[name] = value
		}
		if s.Config.IDColumn != "" && (!idSet || isZeroValue(idValue)) {
			continue
		}
		rows = append(rows, rowWithID{row: rowMap, id: idValue})
	}

	// When an id_column is configured, output rows sorted ascending by its
	// (raw, pre-Format) value.
	if s.Config.IDColumn != "" {
		sort.SliceStable(rows, func(i, j int) bool {
			return compareValues(rows[i].id, rows[j].id) < 0
		})
	}

	result := make([]map[string]any, len(rows))
	for i, r := range rows {
		result[i] = r.row
	}
	return result
}

// compareValues orders two id_column values of the Go types a Cell produces
// (int64, float64, string, bool, time.Time): negative if a < b, positive if
// a > b, 0 if equal or not comparable. int64 and float64 are compared
// numerically against each other, since formatFloat can produce either
// depending on whether the value happens to be a whole number.
func compareValues(a, b any) int {
	if an, ok := asFloat64(a); ok {
		if bn, ok := asFloat64(b); ok {
			switch {
			case an < bn:
				return -1
			case an > bn:
				return 1
			default:
				return 0
			}
		}
	}

	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
	case bool:
		if bv, ok := b.(bool); ok {
			switch {
			case av == bv:
				return 0
			case !av:
				return -1
			default:
				return 1
			}
		}
	case time.Time:
		if bv, ok := b.(time.Time); ok {
			switch {
			case av.Before(bv):
				return -1
			case av.After(bv):
				return 1
			default:
				return 0
			}
		}
	}
	return 0
}

// asFloat64 reports whether v is a numeric Cell value (int64 or float64)
// and, if so, returns it as a float64.
func asFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case int64:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

// isZeroValue reports whether v is the zero value for the Go type a Cell
// produces (int64, float64, string, bool, or time.Time), or is nil (a
// missing value, or one that didn't match its configured column type).
func isZeroValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case int64:
		return t == 0
	case float64:
		return t == 0
	case string:
		return t == ""
	case bool:
		return !t
	case time.Time:
		return t.IsZero()
	default:
		return false
	}
}

func (s Sheet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.marshal())
}

func (s Sheet) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(s.marshal())
}
