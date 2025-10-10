package ssk_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
	"github.com/hashicorp/vault/api"
)

func TestVaultAPICompatibility(t *testing.T) {
	// Setup mock server with mock cipher
	cipher := &mockCipher{}
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /v1/transit/encrypt/{key_id}", ssk.EncryptHandlerFunc(cipher))
	mux.HandleFunc("PUT /v1/transit/decrypt/{key_id}", ssk.DecryptHandlerFunc(cipher))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create Vault client pointing to our mock server
	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	logical := client.Logical()
	keyID := "test-key-123"
	plaintext := []byte("Hello, Vault!")

	// Test encryption
	encryptPath := "transit/encrypt/" + keyID
	encryptData := map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	encryptResp, err := logical.WriteWithContext(context.Background(), encryptPath, encryptData)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encryptResp == nil || encryptResp.Data == nil {
		t.Fatal("encryption response is nil")
	}

	ciphertext, ok := encryptResp.Data["ciphertext"].(string)
	if !ok {
		t.Fatalf("ciphertext not found in response: %+v", encryptResp.Data)
	}

	expectedCiphertext := ssk.VaultPrefix + base64.StdEncoding.EncodeToString(plaintext)
	if ciphertext != expectedCiphertext {
		t.Errorf("ciphertext = %q, want %q", ciphertext, expectedCiphertext)
	}

	// Test decryption
	decryptPath := "transit/decrypt/" + keyID
	decryptData := map[string]any{
		"ciphertext": ciphertext,
	}

	decryptResp, err := logical.WriteWithContext(context.Background(), decryptPath, decryptData)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decryptResp == nil || decryptResp.Data == nil {
		t.Fatal("decryption response is nil")
	}

	// Vault API returns plaintext as base64-encoded string
	decryptedBase64, ok := decryptResp.Data["plaintext"].(string)
	if !ok {
		t.Fatalf("plaintext not found in response: %+v", decryptResp.Data)
	}

	decryptedPlaintext, err := base64.StdEncoding.DecodeString(decryptedBase64)
	if err != nil {
		t.Fatalf("failed to decode base64 plaintext: %v", err)
	}

	if string(decryptedPlaintext) != string(plaintext) {
		t.Errorf("decrypted plaintext = %q, want %q", decryptedPlaintext, plaintext)
	}
}

func TestVaultAPICompatibilityWithBase64(t *testing.T) {
	// Setup mock server with mock cipher
	cipher := &mockCipher{}
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /v1/transit/encrypt/{key_id}", ssk.EncryptHandlerFunc(cipher))
	mux.HandleFunc("PUT /v1/transit/decrypt/{key_id}", ssk.DecryptHandlerFunc(cipher))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create Vault client pointing to our mock server
	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	logical := client.Logical()
	keyID := "test-key-456"
	plaintext := "Hello, Vault with Base64!"

	// Test encryption with base64-encoded plaintext (common usage pattern)
	encryptPath := "transit/encrypt/" + keyID
	encryptData := map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	}

	encryptResp, err := logical.WriteWithContext(context.Background(), encryptPath, encryptData)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encryptResp == nil || encryptResp.Data == nil {
		t.Fatal("encryption response is nil")
	}

	ciphertext, ok := encryptResp.Data["ciphertext"].(string)
	if !ok {
		t.Fatalf("ciphertext not found in response: %+v", encryptResp.Data)
	}

	// Test decryption
	decryptPath := "transit/decrypt/" + keyID
	decryptData := map[string]any{
		"ciphertext": ciphertext,
	}

	decryptResp, err := logical.WriteWithContext(context.Background(), decryptPath, decryptData)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decryptResp == nil || decryptResp.Data == nil {
		t.Fatal("decryption response is nil")
	}

	// Vault API returns plaintext as base64-encoded string
	decryptedBase64, ok := decryptResp.Data["plaintext"].(string)
	if !ok {
		t.Fatalf("plaintext not found in response: %+v", decryptResp.Data)
	}

	// Decode to get original plaintext
	decoded, err := base64.StdEncoding.DecodeString(decryptedBase64)
	if err != nil {
		t.Fatalf("failed to decode base64 plaintext: %v", err)
	}

	if string(decoded) != plaintext {
		t.Errorf("decrypted plaintext = %q, want %q", decoded, plaintext)
	}
}

func TestVaultAPIErrorHandling(t *testing.T) {
	// Setup mock server with mock cipher
	cipher := &mockCipher{}
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /v1/transit/decrypt/{key_id}", ssk.DecryptHandlerFunc(cipher))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create Vault client pointing to our mock server
	config := api.DefaultConfig()
	config.Address = server.URL
	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}

	logical := client.Logical()

	// Test decryption with invalid ciphertext (missing vault prefix)
	decryptPath := "transit/decrypt/test-key"
	decryptData := map[string]any{
		"ciphertext": "invalid-ciphertext-without-prefix",
	}

	_, err = logical.WriteWithContext(context.Background(), decryptPath, decryptData)
	if err == nil {
		t.Error("expected error for invalid ciphertext, got nil")
	}
}
