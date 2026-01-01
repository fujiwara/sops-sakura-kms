package ssk

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var Version = "v0.3.1"

func ShowVersion(ctx context.Context, w io.Writer) (int, error) {
	// Ensure self Version is printed even if error occurs later
	defer fmt.Fprintf(w, "sops-sakura-kms version %s\n", Version)

	os.Setenv("SAKURA_KMS_KEY_ID", "dummy") // dummy value to pass LoadEnv
	env, err := LoadEnv()
	if err != nil {
		return ExitCodeError, fmt.Errorf("failed to load environment variables: %w", err)
	}
	cmd := exec.CommandContext(ctx, env.Command, "--version", "--disable-version-check")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ExitCodeError, fmt.Errorf("failed to execute %s --version: %w", env.Command, err)
	}
	w.Write(output)
	return 0, nil
}
