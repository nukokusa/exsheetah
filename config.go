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
	Sheets   []*SheetConfig `yaml:"sheets"`
	Timezone string         `yaml:"timezone,omitempty"`
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

func (c *Config) Location() (*time.Location, error) {
	if c.Timezone == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(c.Timezone)
}

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
	Name    string          `yaml:"name"`
	Sheet   string          `yaml:"sheet,omitempty"`
	Range   string          `yaml:"range,omitempty"`
	Columns []*ColumnConfig `yaml:"columns"`
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

	return nil
}

func (sc *SheetConfig) SheetName() string {
	if sc.Sheet != "" {
		return sc.Sheet
	}
	return sc.Name
}

type ColumnConfig struct {
	Name string     `yaml:"name"`
	Type ColumnType `yaml:"type"`
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
