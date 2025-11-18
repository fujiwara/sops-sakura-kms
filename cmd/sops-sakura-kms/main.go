package main

import (
	"context"
	"fmt"
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
	args := os.Args[1:]

	// Handle --version flag
	newArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--version" || arg == "-version" {
			fmt.Printf("sops-sakura-kms version %s\n", app.Version)
			return nil
		}
		newArgs = append(newArgs, arg)
	}

	return app.RunWrapper(ctx, newArgs)
}
