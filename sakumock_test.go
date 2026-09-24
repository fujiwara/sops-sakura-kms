package ssk_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/vault/api"
	"github.com/sacloud/sacloud-sdk-go/common/saclient"
	"github.com/sacloud/sakumock/kms"
	"gopkg.in/yaml.v3"
)

// sakumockKeyID is a fixed KMS key ID pre-created in the sakumock server.
const sakumockKeyID = "123456789012"

// newSakumockKMS starts an in-process sakumock KMS server with a fixed key
// and returns its base URL. The server is closed when the test finishes.
func newSakumockKMS(t *testing.T) string {
	t.Helper()
	srv := kms.NewTestServer(kms.Config{
		Keys: map[string]string{sakumockKeyID: "sops-sakura-kms-test-secret"},
	})
	t.Cleanup(srv.Close)
	return srv.TestURL()
}

// sakumockEnv returns environment variables that point sacloud-sdk-go at the given sakumock server.
func sakumockEnv(serverURL string) []string {
	return []string{
		"SAKURA_ENDPOINTS_KMS=" + serverURL,
		"SAKURA_ACCESS_TOKEN=dummy",
		"SAKURA_ACCESS_TOKEN_SECRET=dummy",
	}
}

// newSakumockClient returns a saclient configured for the given sakumock server.
func newSakumockClient(t *testing.T, serverURL string) saclient.ClientAPI {
	t.Helper()
	var sc saclient.Client
	if err := sc.SetEnviron(sakumockEnv(serverURL)); err != nil {
		t.Fatalf("failed to configure saclient: %v", err)
	}
	return &sc
}

// freeAddr returns a 127.0.0.1 address with a free TCP port.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestSakumockEncryptDecrypt(t *testing.T) {
	serverURL := newSakumockKMS(t)
	c, err := ssk.NewSakuraKMSWithClient(newSakumockClient(t, serverURL))
	if err != nil {
		t.Fatalf("failed to create SakuraKMS: %v", err)
	}

	plaintext := []byte("Hello, sakumock!")
	ciphertext, err := c.Encrypt(t.Context(), sakumockKeyID, plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if ciphertext == "" || ciphertext == string(plaintext) {
		t.Fatalf("unexpected ciphertext: %q", ciphertext)
	}

	decrypted, err := c.Decrypt(t.Context(), sakumockKeyID, ciphertext)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}

	if _, err := c.Decrypt(t.Context(), "999999999999", ciphertext); err == nil {
		t.Error("decrypt with unknown key ID should fail")
	}
}

// TestSakumockUnauthorized verifies that a 401 from KMS is propagated to the
// Vault API client as 401 (not 500), so the client does not retry.
func TestSakumockUnauthorized(t *testing.T) {
	srv := kms.NewTestServer(kms.Config{
		Keys:  map[string]string{sakumockKeyID: "sops-sakura-kms-test-secret"},
		Fault: []string{"401:1"},
	})
	t.Cleanup(srv.Close)
	addr := freeAddr(t)

	addEnv, shutdown, err := ssk.RunServer(t.Context(), addr, sakumockKeyID, ssk.WithClient(newSakumockClient(t, srv.TestURL())))
	if err != nil {
		t.Fatalf("RunServer failed: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	config := api.DefaultConfig()
	config.Address = addEnv["VAULT_ADDR"]
	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}
	client.SetToken(addEnv["VAULT_TOKEN"])

	tests := []struct {
		path string
		data map[string]any
	}{
		{"transit/encrypt/" + sakumockKeyID, map[string]any{"plaintext": base64.StdEncoding.EncodeToString([]byte("foo"))}},
		{"transit/decrypt/" + sakumockKeyID, map[string]any{"ciphertext": ssk.VaultPrefix + "Zm9v"}},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			_, err := client.Logical().WriteWithContext(t.Context(), tt.path, tt.data)
			respErr, ok := errors.AsType[*api.ResponseError](err)
			if !ok {
				t.Fatalf("expected *api.ResponseError, got %v", err)
			}
			if respErr.StatusCode != http.StatusUnauthorized {
				t.Errorf("status code = %d, want %d", respErr.StatusCode, http.StatusUnauthorized)
			}
		})
	}
}

func TestSakumockRunServer(t *testing.T) {
	serverURL := newSakumockKMS(t)
	addr := freeAddr(t)

	addEnv, shutdown, err := ssk.RunServer(t.Context(), addr, sakumockKeyID, ssk.WithClient(newSakumockClient(t, serverURL)))
	if err != nil {
		t.Fatalf("RunServer failed: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	wantURI := "http://" + addr + "/v1/transit/encrypt/" + sakumockKeyID
	if got := addEnv["SOPS_VAULT_URIS"]; got != wantURI {
		t.Errorf("SOPS_VAULT_URIS = %q, want %q", got, wantURI)
	}

	config := api.DefaultConfig()
	config.Address = addEnv["VAULT_ADDR"]
	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("failed to create vault client: %v", err)
	}
	client.SetToken(addEnv["VAULT_TOKEN"])
	logical := client.Logical()

	plaintext := []byte("Hello, Vault via sakumock!")
	encryptResp, err := logical.WriteWithContext(t.Context(), "transit/encrypt/"+sakumockKeyID, map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	})
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	ciphertext, ok := encryptResp.Data["ciphertext"].(string)
	if !ok {
		t.Fatalf("ciphertext not found in response: %+v", encryptResp.Data)
	}
	if !strings.HasPrefix(ciphertext, ssk.VaultPrefix) {
		t.Errorf("ciphertext %q does not have prefix %q", ciphertext, ssk.VaultPrefix)
	}

	decryptResp, err := logical.WriteWithContext(t.Context(), "transit/decrypt/"+sakumockKeyID, map[string]any{
		"ciphertext": ciphertext,
	})
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	decryptedBase64, ok := decryptResp.Data["plaintext"].(string)
	if !ok {
		t.Fatalf("plaintext not found in response: %+v", decryptResp.Data)
	}
	decrypted, err := base64.StdEncoding.DecodeString(decryptedBase64)
	if err != nil {
		t.Fatalf("failed to decode plaintext: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

// TestSakumockSOPS runs the real sops binary through RunWrapper against sakumock.
func TestSakumockSOPS(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops binary is not found in PATH")
	}
	serverURL := newSakumockKMS(t)
	for _, kv := range sakumockEnv(serverURL) {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
	t.Setenv("SAKURA_KMS_KEY_ID", sakumockKeyID)
	t.Setenv("SSK_SERVER_ADDR", freeAddr(t))
	t.Setenv("SSK_COMMAND", "sops")

	original := []byte("foo:\n  bar: \"BAR\"\n  description: \"This is bar\"\n")
	dir := t.TempDir()
	plain := filepath.Join(dir, "test.yaml")
	encrypted := filepath.Join(dir, "test.enc.yaml")
	decrypted := filepath.Join(dir, "test.dec.yaml")
	if err := os.WriteFile(plain, original, 0o600); err != nil {
		t.Fatalf("failed to write plaintext file: %v", err)
	}

	exitCode, err := ssk.RunWrapper(t.Context(), []string{"-e", "--output", encrypted, plain})
	if err != nil {
		t.Fatalf("sops -e failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("sops -e exit code = %d, want 0", exitCode)
	}
	encBytes, err := os.ReadFile(encrypted)
	if err != nil {
		t.Fatalf("failed to read encrypted file: %v", err)
	}
	if !strings.Contains(string(encBytes), "ENC[AES256_GCM,") {
		t.Errorf("encrypted file does not contain encrypted values:\n%s", encBytes)
	}
	if !strings.Contains(string(encBytes), "hc_vault:") {
		t.Errorf("encrypted file does not contain hc_vault metadata:\n%s", encBytes)
	}

	exitCode, err = ssk.RunWrapper(t.Context(), []string{"-d", "--output", decrypted, encrypted})
	if err != nil {
		t.Fatalf("sops -d failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("sops -d exit code = %d, want 0", exitCode)
	}
	decBytes, err := os.ReadFile(decrypted)
	if err != nil {
		t.Fatalf("failed to read decrypted file: %v", err)
	}
	// sops re-serializes YAML on output (e.g. drops quotes), so compare parsed values.
	if diff := cmp.Diff(parseYAML(t, original), parseYAML(t, decBytes)); diff != "" {
		t.Errorf("decrypted content mismatch (-want +got):\n%s", diff)
	}
}

func parseYAML(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if err := yaml.Unmarshal(b, &v); err != nil {
		t.Fatalf("failed to parse YAML: %v\n%s", err, b)
	}
	return v
}
