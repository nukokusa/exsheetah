package exsheetah

import (
	"context"
)

var Version string

func New(ctx context.Context) (*CLI, error) {
	c := &CLI{}
	return c, nil
}
