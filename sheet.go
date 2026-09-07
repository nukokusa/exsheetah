package exsheetah

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/samber/lo"
)

type Sheet struct {
	Config  *SheetConfig
	Columns []string
	Rows    [][]Cell
}

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
	columnTypeMap := lo.SliceToMap(s.Config.Columns, func(c *ColumnConfig) (string, ColumnType) {
		return c.Name, c.Type
	})

	headers := make(map[int]string)
	for i, column := range s.Columns {
		headers[i] = column
	}

	var result []map[string]any
	for _, row := range s.Rows {
		rowMap := make(map[string]any)
		for i, cell := range row {
			name, ok := headers[i]
			if !ok {
				continue
			}
			column, ok := columnTypeMap[name]
			if !ok {
				continue
			}
			if value := cell.Value(column); value != nil {
				rowMap[name] = value
			}
		}
		result = append(result, rowMap)
	}
	return result
}

func (s Sheet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.marshal())
}

func (s Sheet) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(s.marshal())
}
