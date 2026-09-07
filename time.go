package exsheetah

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

func ParseTimeByString(str string, loc *time.Location) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05Z07:00", // RFC3339
		"2006-01-02 15:04:05Z07:00",
		"2006/01/02T15:04:05Z07:00",
		"2006/01/02 15:04:05Z07:00",
	}
	for _, format := range formats {
		parsed, err := time.Parse(format, str)
		if err == nil {
			return parsed, nil
		}
	}

	formats = []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02T15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	}
	for _, format := range formats {
		parsed, err := time.ParseInLocation(format, str, loc)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse time: %s", str)
}

// TimeFromExcelSerial converts an xlsx serial date/time number into a
// time.Time. The result's wall-clock components (year, month, day, hour,
// minute, second) are preserved as-is and simply re-anchored to loc, since
// xlsx serial numbers carry no timezone information of their own.
func TimeFromExcelSerial(serial float64, date1904 bool, loc *time.Location) (time.Time, error) {
	t, err := excelize.ExcelDateToTime(serial, date1904)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
}
