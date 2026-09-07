package exsheetah

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = cerr
		}
	}()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

type Config struct {
	// Timezone is the IANA timezone name (e.g. "Asia/Tokyo") used to
	// interpret date/time cell values, since xlsx stores them with no
	// timezone of their own. Defaults to UTC when omitted.
	Timezone string         `yaml:"timezone,omitempty"`
	Sheets   []*SheetConfig `yaml:"sheets"`
}

func (c *Config) Validate() error {
	if len(c.Sheets) == 0 {
		return errors.New("sheets is empty")
	}
	if _, err := c.Location(); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	for _, sc := range c.Sheets {
		if err := sc.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Location returns the time.Location described by Timezone, defaulting to
// UTC when Timezone is empty.
func (c *Config) Location() (*time.Location, error) {
	if c.Timezone == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(c.Timezone)
}

// FilterSheets returns the configured sheets whose Name is in names, kept
// in the configuration file's original order. When names is empty, every
// configured sheet is returned. It is an error for names to contain a name
// that does not match any configured sheet.
func (c *Config) FilterSheets(names []string) ([]*SheetConfig, error) {
	if len(names) == 0 {
		return c.Sheets, nil
	}

	want := make(map[string]bool, len(names))
	for _, name := range names {
		want[name] = true
	}

	var filtered []*SheetConfig
	for _, sc := range c.Sheets {
		if want[sc.Name] {
			filtered = append(filtered, sc)
			delete(want, sc.Name)
		}
	}

	if len(want) > 0 {
		missing := make([]string, 0, len(want))
		for name := range want {
			missing = append(missing, name)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("sheet(s) not found in config: %s", strings.Join(missing, ", "))
	}

	return filtered, nil
}

type SheetConfig struct {
	// Name is both the configuration entry name and, unless Sheet is set,
	// the name of the worksheet inside the xlsx file to read from.
	Name string `yaml:"name"`
	// Sheet optionally overrides the worksheet name to read from, in case
	// it should differ from Name.
	Sheet string `yaml:"sheet,omitempty"`
	// Range is an A1-style cell range (e.g. "A1:D10") restricting which
	// cells are read. When empty, the sheet's whole used range is read.
	Range   string          `yaml:"range,omitempty"`
	Columns []*ColumnConfig `yaml:"columns"`
	// IDColumn optionally names one of Columns whose value identifies the
	// row. When set, a row whose IDColumn value is the zero value for its
	// column type (0, "", false, a zero time, or the value is missing or
	// doesn't match the column type) is excluded from the output.
	IDColumn string `yaml:"id_column,omitempty"`
}

var a1Regex = regexp.MustCompile(`^([A-Z]+[0-9]+)(:[A-Z]+[0-9]+)?$`)

func (sc *SheetConfig) Validate() error {
	if sc.Name == "" {
		return errors.New("name is empty")
	}
	if sc.Range != "" {
		if !a1Regex.MatchString(sc.Range) {
			return fmt.Errorf("invalid range format: %s", sc.Range)
		}
	}
	if len(sc.Columns) == 0 {
		return errors.New("columns is empty")
	}
	for _, cc := range sc.Columns {
		if err := cc.Validate(); err != nil {
			return err
		}
	}
	if sc.IDColumn != "" {
		found := false
		for _, cc := range sc.Columns {
			if cc.Name == sc.IDColumn {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("id_column not found in columns: %s", sc.IDColumn)
		}
	}

	return nil
}

// SheetName returns the name of the worksheet to read from the xlsx file.
func (sc *SheetConfig) SheetName() string {
	if sc.Sheet != "" {
		return sc.Sheet
	}
	return sc.Name
}

type ColumnConfig struct {
	Name string     `yaml:"name"`
	Type ColumnType `yaml:"type"`
	// Format is a Go reference-time layout (e.g. "2006-01-02" or
	// time.RFC3339) used to render a timestamp column's value in the
	// output. Only valid when Type is "timestamp"; when omitted, the
	// value is output as a plain RFC 3339 timestamp.
	Format string `yaml:"format,omitempty"`
}

func (cc *ColumnConfig) Validate() error {
	if cc.Name == "" {
		return errors.New("column name is required")
	}
	if cc.Type == "" {
		return errors.New("column type is required")
	}
	if err := cc.Type.Validate(); err != nil {
		return err
	}
	if cc.Format != "" && cc.Type != ColumnTypeTimestamp {
		return fmt.Errorf("format is only valid for timestamp columns: %s", cc.Name)
	}

	return nil
}

type ColumnType string

const (
	ColumnTypeString    ColumnType = "string"
	ColumnTypeNumber    ColumnType = "number"
	ColumnTypeBool      ColumnType = "boolean"
	ColumnTypeTimestamp ColumnType = "timestamp"
)

func (ct ColumnType) Validate() error {
	switch ct {
	case ColumnTypeString, ColumnTypeNumber, ColumnTypeBool, ColumnTypeTimestamp:
		return nil
	default:
		return fmt.Errorf("not supported column type: %s", ct)
	}
}
