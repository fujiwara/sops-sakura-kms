package ssk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	ssk "github.com/fujiwara/sops-sakura-kms"
	"github.com/getsops/sops/v3/decrypt"
)

// ExampleRunServer demonstrates how to use RunServer with the SOPS decrypt package
// to decrypt SOPS-encrypted files in your Go application.
func ExampleRunServer() {
	ctx := context.Background()

	// Start Vault Transit Engine compatible server
	// Requires SAKURACLOUD_ACCESS_TOKEN and SAKURACLOUD_ACCESS_TOKEN_SECRET env vars
	addEnv, shutdown, err := ssk.RunServer(ctx, "127.0.0.1:8200", os.Getenv("SAKURACLOUD_KMS_KEY_ID"))
	if err != nil {
		panic(err)
	}
	defer shutdown(context.Background())

	// Set environment variables for SOPS library
	for k, v := range addEnv {
		os.Setenv(k, v)
	}

	// Decrypt SOPS-encrypted file using SOPS library
	plaintext, err := decrypt.File("secrets.enc.yaml", "yaml")
	if err != nil {
		panic(err)
	}

	// Parse decrypted content
	var config map[string]string
	if err := json.Unmarshal(plaintext, &config); err != nil {
		panic(err)
	}

	fmt.Println("Decrypted successfully")
}
