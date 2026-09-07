package exsheetah

import (
	"math"
	"strconv"
	"time"
)

type Cell interface {
	Value(typ ColumnType) any
}

type StringCell struct {
	str string
	loc *time.Location
}

func NewStringCell(str string, loc *time.Location) *StringCell {
	return &StringCell{str, loc}
}

func (c *StringCell) Value(typ ColumnType) any {
	switch typ {
	case ColumnTypeString:
		return c.str
	case ColumnTypeNumber:
		if num, err := strconv.ParseFloat(c.str, 64); err == nil {
			return formatFloat(num)
		}
		return nil
	case ColumnTypeBool:
		if b, err := strconv.ParseBool(c.str); err == nil {
			return b
		}
		return nil
	case ColumnTypeTimestamp:
		if t, err := ParseTimeByString(c.str, c.loc); err == nil {
			return t
		}
		return nil
	default:
		return nil
	}
}

type NumberCell float64

func (c NumberCell) Value(typ ColumnType) any {
	switch typ {
	case ColumnTypeNumber:
		return formatFloat(float64(c))
	case ColumnTypeString:
		return strconv.FormatFloat(float64(c), 'f', -1, 64)
	case ColumnTypeBool:
		if c != 0 {
			return true
		}
		return false
	default:
		return nil
	}
}

type BoolCell bool

func (c BoolCell) Value(typ ColumnType) any {
	switch typ {
	case ColumnTypeBool:
		return bool(c)
	case ColumnTypeString:
		return strconv.FormatBool(bool(c))
	case ColumnTypeNumber:
		if bool(c) {
			return int64(1)
		}
		return int64(0)
	default:
		return nil
	}
}

type NilCell struct{}

func (c NilCell) Value(typ ColumnType) any {
	return nil
}

func formatFloat(v float64) any {
	if num := math.Trunc(v); num == v {
		return int64(num)
	}
	return v
}
