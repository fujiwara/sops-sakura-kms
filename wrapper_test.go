package ssk_test

import (
	"context"
	"os"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

func TestRunWrapper_WithoutKeyID(t *testing.T) {
	// Ensure SAKURA_KMS_KEY_ID is not set
	os.Unsetenv(ssk.EnvKeyID)

	err := ssk.RunWrapper(context.Background(), []string{"--version"})
	if err == nil {
		t.Error("expected error when SAKURA_KMS_KEY_ID is not set, got nil")
	}

	expectedMsg := ssk.EnvKeyID + " environment variable is required"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestRunWrapper_WithHcVaultTransitArg(t *testing.T) {
	// Set SAKURA_KMS_KEY_ID for this test
	os.Setenv(ssk.EnvKeyID, "test-key")
	defer os.Unsetenv(ssk.EnvKeyID)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "with --hc-vault-transit flag",
			args: []string{"--hc-vault-transit", "http://example.com/v1/transit/keys/my-key", "test.yaml"},
		},
		{
			name: "with --hc-vault-transit= format",
			args: []string{"--hc-vault-transit=http://example.com/v1/transit/keys/my-key", "test.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ssk.RunWrapper(context.Background(), tt.args)
			if err == nil {
				t.Error("expected error when --hc-vault-transit is specified, got nil")
			}

			expectedMsg := "--hc-vault-transit should not be specified when using this wrapper; it will be set automatically from " + ssk.EnvKeyID
			if err.Error() != expectedMsg {
				t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
			}
		})
	}
}
