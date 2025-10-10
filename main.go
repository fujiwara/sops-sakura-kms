package ssk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	VaultPrefix    = "vault:v1:"
	KeyIDPathParam = "key_id"
	EnvKeyID       = "SAKURA_KMS_KEY_ID"
)

func NewMux(cipher Cipher) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthCheckHandler)
	mux.HandleFunc(makeURL("PUT /v1/transit/encrypt/{%s}"), EncryptHandlerFunc(cipher))
	mux.HandleFunc(makeURL("PUT /v1/transit/decrypt/{%s}"), DecryptHandlerFunc(cipher))
	return mux
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func RunWrapper(ctx context.Context, sopsArgs []string) error {
	keyID := os.Getenv(EnvKeyID)
	if keyID == "" {
		return fmt.Errorf("%s environment variable is required", EnvKeyID)
	}

	// Check if --hc-vault-transit is already specified
	for _, arg := range sopsArgs {
		if arg == "--hc-vault-transit" || strings.HasPrefix(arg, "--hc-vault-transit=") {
			return fmt.Errorf("--hc-vault-transit should not be specified when using this wrapper; it will be set automatically from %s", EnvKeyID)
		}
	}

	// Prepend --hc-vault-transit argument
	vaultTransitURI := fmt.Sprintf("http://127.0.0.1:8200/v1/transit/encrypt/%s", keyID)
	sopsArgs = append([]string{"--hc-vault-transit", vaultTransitURI}, sopsArgs...)

	slog.Info("Starting Vault-compatible API server for Sakura KMS", "key_id", keyID, "addr", "127.0.0.1:8200")

	// 1. Create and start server
	cipher, err := NewSakuraKMS()
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	mux := NewMux(cipher)
	server := &http.Server{Addr: "127.0.0.1:8200", Handler: mux}

	go server.ListenAndServe()
	defer server.Shutdown(context.Background())

	// 2. Wait for server to become healthy
	if err := waitForServer(ctx, "http://127.0.0.1:8200/health"); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	slog.Info("Server started successfully, executing SOPS", "command", "sops", "args", sopsArgs)

	// 3. Set environment variables for SOPS
	env := append(os.Environ(),
		"VAULT_ADDR=http://127.0.0.1:8200",
		"VAULT_TOKEN=dummy",
	)

	// 4. Execute SOPS command
	cmd := exec.CommandContext(ctx, "sops", sopsArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func waitForServer(ctx context.Context, healthURL string) error {
	client := &http.Client{Timeout: 100 * time.Millisecond}
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server did not become healthy")
}

func Run(ctx context.Context) error {
	cipher, err := NewSakuraKMS()
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}
	mux := NewMux(cipher)
	sv := &http.Server{
		Addr:    "127.0.0.1:8200",
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		slog.Info("shutting down server...")
		sv.Shutdown(context.Background())
	}()
	slog.Info("starting server...", "addr", sv.Addr)
	return sv.ListenAndServe()
}

func makeURL(path string) string {
	return fmt.Sprintf(path, KeyIDPathParam)
}

func errorResponse(w http.ResponseWriter, err error, status int) {
	slog.Error("error response", "status", status, "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{
			{
				"status": status,
				"detail": err.Error(),
			},
		},
	})
}

func EncryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue("key_id")
		slog.Info("Encrypting data with Sakura KMS", "key_id", keyID)
		req := &VaultEncryptRequest{}
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			errorResponse(w, err, http.StatusBadRequest)
			return
		}
		// Decode base64-encoded plaintext
		plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
		if err != nil {
			errorResponse(w, fmt.Errorf("invalid base64 plaintext: %w", err), http.StatusBadRequest)
			return
		}
		ciphertext, err := cipher.Encrypt(r.Context(), keyID, plaintext)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		res := &VaultEncryptResponse{
			Ciphertext: VaultPrefix + ciphertext,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

func DecryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue("key_id")
		slog.Info("Decrypting data with Sakura KMS", "key_id", keyID)
		req := &VaultDecryptRequest{}
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			errorResponse(w, err, http.StatusBadRequest)
			return
		}
		body := strings.TrimPrefix(req.Ciphertext, VaultPrefix)
		if len(body) == len(req.Ciphertext) {
			errorResponse(w, fmt.Errorf("invalid ciphertext format"), http.StatusBadRequest)
			return
		}
		plaintext, err := cipher.Decrypt(r.Context(), keyID, body)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		// Encode plaintext as base64 for response
		res := &VaultDecryptResponse{
			Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}
