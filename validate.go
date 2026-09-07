package exsheetah

import (
	"context"
	"fmt"
)

type ValidateOption struct{}

func (o *ValidateOption) Validate() error {
	return nil
}

func (c *CLI) runValidate(ctx context.Context, opt *ValidateOption) error {
	var err error
	if err = opt.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if _, err = LoadConfig(c.Config); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	return nil
}
