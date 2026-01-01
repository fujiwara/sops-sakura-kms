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
	exitCode, err := run(ctx)
	if err != nil {
		slog.Error(err.Error())
	}
	os.Exit(exitCode)
}

func run(ctx context.Context) (int, error) {
	args := os.Args[1:]

	// Handle --version flag
	newArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--version" || arg == "-version" {
			return app.ShowVersion(ctx, os.Stdout)
		}
		newArgs = append(newArgs, arg)
	}

	return app.RunWrapper(ctx, newArgs)
}
