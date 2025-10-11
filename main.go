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
	EnvKeyID       = "SAKURACLOUD_KMS_KEY_ID"
	ServerAddr     = "127.0.0.1:8200"
	SOPSbin        = "sops"
)

// NewMux creates a new HTTP ServeMux with Vault Transit Engine compatible API endpoints.
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

// newServer creates a new HTTP server with Vault Transit Engine compatible API.
func newServer(cipher Cipher) *http.Server {
	mux := NewMux(cipher)
	return &http.Server{Addr: ServerAddr, Handler: mux}
}

// RunWrapper starts a Vault Transit Engine compatible API server and executes SOPS command.
// It automatically configures SOPS to use Sakura Cloud KMS via SOPS_VAULT_URIS environment variable.
// Requires SAKURA_KMS_KEY_ID environment variable to be set.
func RunWrapper(ctx context.Context, sopsArgs []string) error {
	keyID := os.Getenv(EnvKeyID)
	if keyID == "" {
		return fmt.Errorf("%s environment variable is required", EnvKeyID)
	}

	slog.Info("Starting Vault-compatible API server for Sakura KMS", "key_id", keyID, "addr", ServerAddr)

	// 1. Create cipher
	cipher, err := NewSakuraKMS()
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// 2. Create and start server
	server := newServer(cipher)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()
	defer server.Shutdown(context.Background())

	// 3. Wait for server to become healthy
	if err := waitForServer(ctx, fmt.Sprintf("http://%s/health", ServerAddr)); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	slog.Info("Server started successfully, executing SOPS", "command", SOPSbin, "args", sopsArgs)

	// 4. Set environment variables for SOPS
	vaultTransitURI := fmt.Sprintf("http://%s/v1/transit/encrypt/%s", ServerAddr, keyID)
	env := append(os.Environ(),
		"VAULT_ADDR=http://"+ServerAddr,
		"VAULT_TOKEN=dummy",
		"SOPS_VAULT_URIS="+vaultTransitURI,
	)

	// 5. Execute SOPS command
	cmd := exec.CommandContext(ctx, SOPSbin, sopsArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func waitForServer(ctx context.Context, healthURL string) error {
	interval := 100 * time.Millisecond
	client := &http.Client{Timeout: interval}
	for range 30 {
		req, _ := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("server did not become healthy")
}

// Run starts a Vault Transit Engine compatible API server for Sakura Cloud KMS.
// The server listens on 127.0.0.1:8200 and provides encrypt/decrypt endpoints.
func Run(ctx context.Context) error {
	cipher, err := NewSakuraKMS()
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}
	sv := newServer(cipher)
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

// readRequest decodes JSON request body into the specified type.
// Validates Content-Type header and decodes the request body.
func readRequest[T any](r *http.Request) (*T, error) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("invalid content-type: %s", contentType)
	}
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// jsonResponse writes a JSON response with the given status code and body.
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

func errorResponse(w http.ResponseWriter, err error, status int) {
	slog.Error("error response", "status", status, "error", err)
	res := &VaultErrorResponse{
		Errors: []string{err.Error()},
	}
	jsonResponse(w, status, res)
}

// EncryptHandlerFunc returns an HTTP handler for Vault Transit Engine encrypt endpoint.
func EncryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue(KeyIDPathParam)
		slog.Info("Encrypting data with Sakura KMS", "key_id", keyID)
		req, err := readRequest[VaultEncryptRequest](r)
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
		jsonResponse(w, http.StatusOK, res)
	}
}

// DecryptHandlerFunc returns an HTTP handler for Vault Transit Engine decrypt endpoint.
func DecryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue(KeyIDPathParam)
		slog.Info("Decrypting data with Sakura KMS", "key_id", keyID)
		req, err := readRequest[VaultDecryptRequest](r)
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
		jsonResponse(w, http.StatusOK, res)
	}
}
