package ssk_test

import (
	"os"
	"strconv"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
	"github.com/google/go-cmp/cmp"
)

var envSet = map[string]string{
	"SAKURACLOUD_KMS_KEY_ID": "example-key-id-2",
	"SSK_SERVER_ADDR":        "192.168.0.1:8200",
	"SSK_COMMAND":            "/usr/local/bin/sops",
	"SSK_SERVER_ONLY":        "true",
}

func TestParseEnv(t *testing.T) {
	for k, v := range envSet {
		t.Setenv(k, v)
	}
	e, err := ssk.LoadEnv()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	serverOnly, _ := strconv.ParseBool(os.Getenv("SSK_SERVER_ONLY")) // default is false
	if diff := cmp.Diff(&ssk.Env{
		ServerAddr: os.Getenv("SSK_SERVER_ADDR"),
		Command:    os.Getenv("SSK_COMMAND"),
		KMSKeyID:   os.Getenv("SAKURACLOUD_KMS_KEY_ID"),
		ServerOnly: serverOnly,
	}, e); diff != "" {
		t.Errorf("parsed env mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnvDefault(t *testing.T) {
	t.Setenv("SAKURACLOUD_KMS_KEY_ID", "default-key-id")
	e, err := ssk.LoadEnv()
	if err != nil {
		t.Fatalf("failed to load environment variables: %v", err)
	}
	if diff := cmp.Diff(&ssk.Env{
		ServerAddr: "127.0.0.1:8200",
		Command:    "sops",
		KMSKeyID:   os.Getenv("SAKURACLOUD_KMS_KEY_ID"),
		ServerOnly: false,
	}, e); diff != "" {
		t.Errorf("parsed env mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnvRequired(t *testing.T) {
	t.Setenv("SAKURACLOUD_KMS_KEY_ID", "")
	_, err := ssk.LoadEnv()
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
}

func TestLoadEnv(t *testing.T) {
	t.Run("all values from environment", func(t *testing.T) {
		for k, v := range envSet {
			t.Setenv(k, v)
		}

		env, err := ssk.LoadEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		serverOnly, _ := strconv.ParseBool(envSet["SSK_SERVER_ONLY"])
		if diff := cmp.Diff(&ssk.Env{
			KMSKeyID:   envSet["SAKURACLOUD_KMS_KEY_ID"],
			ServerOnly: serverOnly,
			ServerAddr: envSet["SSK_SERVER_ADDR"],
			Command:    envSet["SSK_COMMAND"],
		}, env); diff != "" {
			t.Errorf("LoadEnv mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("default values", func(t *testing.T) {
		t.Setenv("SAKURACLOUD_KMS_KEY_ID", "test-key-id")

		env, err := ssk.LoadEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diff := cmp.Diff(&ssk.Env{
			KMSKeyID:   "test-key-id",
			ServerOnly: false,
			ServerAddr: "127.0.0.1:8200",
			Command:    "sops",
		}, env); diff != "" {
			t.Errorf("LoadEnv mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("required field missing", func(t *testing.T) {
		t.Setenv("SAKURACLOUD_KMS_KEY_ID", "")
		_, err := ssk.LoadEnv()
		if err == nil {
			t.Fatal("expected error for missing required field, got nil")
		}
	})

	t.Run("invalid boolean value", func(t *testing.T) {
		t.Setenv("SAKURACLOUD_KMS_KEY_ID", "test-key-id")
		t.Setenv("SSK_SERVER_ONLY", "invalid")

		_, err := ssk.LoadEnv()
		if err == nil {
			t.Fatal("expected error for invalid boolean value, got nil")
		}
	})

	t.Run("SAKURA_KMS_KEY_ID only", func(t *testing.T) {
		t.Setenv("SAKURA_KMS_KEY_ID", "sakura-key-id")

		env, err := ssk.LoadEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env.KMSKeyID != "sakura-key-id" {
			t.Errorf("KMSKeyID = %q, want %q", env.KMSKeyID, "sakura-key-id")
		}
	})

	t.Run("SAKURA_KMS_KEY_ID takes priority over SAKURACLOUD_KMS_KEY_ID", func(t *testing.T) {
		t.Setenv("SAKURA_KMS_KEY_ID", "sakura-key-id")
		t.Setenv("SAKURACLOUD_KMS_KEY_ID", "sakuracloud-key-id")

		env, err := ssk.LoadEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env.KMSKeyID != "sakura-key-id" {
			t.Errorf("KMSKeyID = %q, want %q (SAKURA_KMS_KEY_ID should take priority)", env.KMSKeyID, "sakura-key-id")
		}
	})
}
