package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nukokusa/exsheetah"
)

var Version = "current"

func main() {
	exsheetah.Version = Version
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	c, err := exsheetah.New(ctx)
	if err != nil {
		return err
	}
	return c.Run(ctx)
}
