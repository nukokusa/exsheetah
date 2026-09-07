package exsheetah

import (
	"context"
	"fmt"
)

type ExportOption struct {
	File   string   `help:"Path to the xlsx file to export from" required:"" type:"existingfile" name:"file" env:"EXSHEETAH_FILE"`
	Format string   `help:"Export format [yaml,json]" default:"yaml" enum:"yaml,json"`
	Dir    string   `help:"Export directory" type:"path" default:"."`
	Sheets []string `help:"Sheet names (matching a SheetConfig name) to export; exports every configured sheet when omitted" name:"sheets"`
}

func (o *ExportOption) Validate() error {
	return nil
}

func (c *CLI) runExport(ctx context.Context, opt *ExportOption) error {
	var err error
	if err = opt.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	config, err := LoadConfig(c.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	loc, err := config.Location()
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	sheetConfigs, err := config.FilterSheets(opt.Sheets)
	if err != nil {
		return fmt.Errorf("invalid sheets: %w", err)
	}

	reader, err := NewReader(opt.File)
	if err != nil {
		return fmt.Errorf("failed to open xlsx file: %w", err)
	}
	defer reader.Close()

	sheets, err := reader.ReadSheets(sheetConfigs, loc)
	if err != nil {
		return fmt.Errorf("failed to read sheets: %w", err)
	}

	var o Outputter
	switch opt.Format {
	case "yaml":
		o = &YAMLOutputter{}
	case "json":
		o = &JSONOutputter{}
	default:
		return fmt.Errorf("invalid format: %s", opt.Format)
	}

	if err := o.Output(ctx, opt.Dir, sheets...); err != nil {
		return fmt.Errorf("failed to output sheets: %w", err)
	}
	return nil
}
