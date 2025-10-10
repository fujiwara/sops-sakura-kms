package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	app "github.com/fujiwara/sops-sakura-kms"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), signals()...)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	return app.RunWrapper(ctx, os.Args[1:])
}
