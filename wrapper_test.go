package ssk_test

import (
	"context"
	"os"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

func TestRunWrapper_WithoutKeyID(t *testing.T) {
	// Ensure SAKURACLOUD_KMS_KEY_ID is not set
	os.Unsetenv(ssk.EnvKeyID)

	err := ssk.RunWrapper(context.Background(), []string{"--version"})
	if err == nil {
		t.Error("expected error when SAKURACLOUD_KMS_KEY_ID is not set, got nil")
	}

	expectedMsg := ssk.EnvKeyID + " environment variable is required"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}
