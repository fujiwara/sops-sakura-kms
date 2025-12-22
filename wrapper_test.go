package ssk_test

import (
	"context"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

func TestRunWrapperExitCode(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		args         []string
		wantExitCode int
	}{
		{
			name:         "exit 0",
			command:      "sh",
			args:         []string{"-c", "exit 0"},
			wantExitCode: 0,
		},
		{
			name:         "exit 1",
			command:      "sh",
			args:         []string{"-c", "exit 1"},
			wantExitCode: 1,
		},
		{
			name:         "exit 2",
			command:      "sh",
			args:         []string{"-c", "exit 2"},
			wantExitCode: 2,
		},
		{
			name:         "exit 42",
			command:      "sh",
			args:         []string{"-c", "exit 42"},
			wantExitCode: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SAKURACLOUD_KMS_KEY_ID", "test-key-id")
			t.Setenv("SSK_COMMAND", tt.command)

			exitCode, err := ssk.RunWrapper(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exitCode != tt.wantExitCode {
				t.Errorf("exitCode = %d, want %d", exitCode, tt.wantExitCode)
			}
		})
	}
}

func TestRunWrapperMissingKeyID(t *testing.T) {
	t.Setenv("SAKURACLOUD_KMS_KEY_ID", "")

	exitCode, err := ssk.RunWrapper(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for missing key ID, got nil")
	}
	if exitCode != ssk.ExitCodeError {
		t.Errorf("exitCode = %d, want %d", exitCode, ssk.ExitCodeError)
	}
}
