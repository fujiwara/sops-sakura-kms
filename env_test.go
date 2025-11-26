package ssk_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/alecthomas/kong"
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
	var e ssk.Env
	k, err := kong.New(&e)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	k.Parse([]string{})
	serverOnly, _ := strconv.ParseBool(os.Getenv("SSK_SERVER_ONLY")) // default is false
	if diff := cmp.Diff(ssk.Env{
		ServerAddr: os.Getenv("SSK_SERVER_ADDR"),
		Command:    os.Getenv("SSK_COMMAND"),
		KMSKeyID:   os.Getenv("SAKURACLOUD_KMS_KEY_ID"),
		ServerOnly: serverOnly,
	}, e); diff != "" {
		t.Errorf("parsed env mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnvDefault(t *testing.T) {
	var e ssk.Env
	k, err := kong.New(&e)
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	k.Parse([]string{})
	if diff := cmp.Diff(ssk.Env{
		ServerAddr: "127.0.0.1:8200",
		Command:    "sops",
		KMSKeyID:   os.Getenv("SAKURACLOUD_KMS_KEY_ID"),
		ServerOnly: false,
	}, e); diff != "" {
		t.Errorf("parsed env mismatch (-want +got):\n%s", diff)
	}
}
