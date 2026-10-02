package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"app/backend/services/identity/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		slog.Error("service failed", "error", err)
		os.Exit(1)
	}
}
