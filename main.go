package ssk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/sacloud/sacloud-sdk-go/common/saclient"
)

const (
	VaultPrefix    = "vault:v1:"
	KeyIDPathParam = "key_id"

	// DefaultServerAddr is the Vault address recorded in SOPS files when the
	// server listens on an ephemeral port. It is also the listen address in
	// server-only mode when SSK_SERVER_ADDR is not set.
	DefaultServerAddr = "127.0.0.1:8200"

	// ephemeralServerAddr is the listen address used by the wrapper when
	// SSK_SERVER_ADDR is not set, so that multiple processes can run at once.
	ephemeralServerAddr = "127.0.0.1:0"

	// ExitCodeError is the exit code returned when an error occurs in the application.
	ExitCodeError = 1
)

// IsStdinTerminal reports whether stdin is a terminal. It is a
// package variable so tests can substitute their own check; production
// callers should leave it alone.
var IsStdinTerminal = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// NewMux creates a new HTTP ServeMux with Vault Transit Engine compatible API endpoints.
func NewMux(cipher Cipher) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthCheckHandler)
	mux.HandleFunc("PUT /v1/transit/encrypt/{key_id}", EncryptHandlerFunc(cipher))
	mux.HandleFunc("PUT /v1/transit/decrypt/{key_id}", DecryptHandlerFunc(cipher))
	return mux
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// newServer creates a new HTTP server with Vault Transit Engine compatible API.
func newServer(cipher Cipher) *http.Server {
	mux := NewMux(cipher)
	return &http.Server{Handler: mux}
}

// RunWrapper starts a Vault Transit Engine compatible API server and executes a command.
// It automatically configures SOPS to use Sakura Cloud KMS via SOPS_VAULT_URIS environment variable.
// Requires SAKURA_KMS_KEY_ID environment variable to be set.
// Returns the exit code of the executed command and any error that occurred.
func RunWrapper(ctx context.Context, args []string) (int, error) {
	e, err := LoadEnv()
	if err != nil {
		return ExitCodeError, fmt.Errorf("failed to load environment variables: %w", err)
	}
	slog.Debug("Parsed command-line arguments", "env", e)

	// Start server
	addEnv, shutdown, err := RunServer(ctx, e.listenAddr(), e.KMSKeyID)
	if err != nil {
		return ExitCodeError, fmt.Errorf("failed to start server: %w", err)
	}
	defer shutdown(context.Background())
	slog.Info("Started Vault-compatible API server for Sakura KMS", "key_id", e.KMSKeyID, "addr", addEnv["VAULT_AGENT_ADDR"])

	if e.ServerOnly {
		slog.Info("Server is running in server-only mode")
		<-ctx.Done()
		return 0, nil
	}

	slog.Info("Server started successfully, executing", "command", e.Command, "args", args)

	// Execute command. `sops exec-env` from a tty typically launches an
	// interactive program (mysql cli, editor, ...) that wants to handle
	// Ctrl-C on its own — and possibly receive many of them. SIGINT
	// reaches the child via the foreground process group, so we must
	// not pass ctx to exec there (CommandContext would SIGKILL the
	// child the moment our own ctx is cancelled by the inherited
	// SIGINT).
	//
	// Everywhere else we keep exec.CommandContext but override its
	// Cancel to send SIGTERM rather than the default SIGKILL, with a
	// short WaitDelay as the SIGKILL escalation grace period. This
	// lets the child (sops itself, an editor under `sops -i`, a batch
	// process under non-tty `exec-env`, ...) clean up before exiting.
	isExecEnv := slices.Contains(args, "exec-env")
	var cmd *exec.Cmd
	if isExecEnv && IsStdinTerminal() {
		cmd = exec.Command(e.Command, args...)
	} else {
		cmd = exec.CommandContext(ctx, e.Command, args...)
		cmd.Cancel = func() error {
			return cmd.Process.Signal(syscall.SIGTERM)
		}
		cmd.WaitDelay = 5 * time.Second
	}
	cmd.Env = os.Environ()
	for k, v := range addEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if e.KMSKeyID == "" {
			slog.Warn("command exited with error. If you need to encrypt, set SAKURA_KMS_KEY_ID or configure hc_vault_transit_uri in .sops.yaml")
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return exitErr.ExitCode(), nil
		}
		return ExitCodeError, err
	}
	return 0, nil
}

// Option is a functional option for RunServer.
type Option func(*serverOptions)

type serverOptions struct {
	cipher Cipher
	client saclient.ClientAPI
}

// WithCipher sets a custom Cipher implementation. Useful for testing.
func WithCipher(c Cipher) Option {
	return func(o *serverOptions) {
		o.cipher = c
	}
}

// WithClient sets a saclient.ClientAPI for creating the KMS cipher.
// This takes precedence over the default environment variable-based client.
func WithClient(c saclient.ClientAPI) Option {
	return func(o *serverOptions) {
		o.client = c
	}
}

// RunServer starts the Vault Transit Engine compatible API server listening on addr.
//
// If the port of addr is 0, the server listens on an ephemeral port, and
// SOPS_VAULT_URIS points to DefaultServerAddr so that the address recorded in
// SOPS files does not depend on the port. VAULT_AGENT_ADDR is always set to
// the actual listen address (with a loopback host if the host of addr is empty
// or unspecified); the Vault API client used by SOPS connects to it
// instead of the address recorded in SOPS files. This allows multiple servers
// to run on the same host at once.
//
// Without options, it uses Sakura Cloud KMS with credentials from environment variables.
// Use WithCipher to provide a custom cipher, or WithClient to provide a pre-configured saclient.
// Returns environment variables to configure SOPS, a shutdown function, and any error that occurred.
func RunServer(ctx context.Context, addr, keyID string, opts ...Option) (map[string]string, func(context.Context) error, error) {
	var o serverOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.cipher == nil {
		var (
			cipher Cipher
			err    error
		)
		if o.client != nil {
			cipher, err = NewSakuraKMSWithClient(o.client)
		} else {
			cipher, err = NewSakuraKMS()
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
		}
		o.cipher = cipher
	}
	return runServer(ctx, addr, keyID, o.cipher)
}

func runServer(_ context.Context, addr, keyID string, cipher Cipher) (map[string]string, func(context.Context) error, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid server address %q: %w", addr, err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// The address recorded in SOPS files (SOPS_VAULT_URIS).
	fileAddr := addr
	if port == "0" {
		fileAddr = DefaultServerAddr
		_, port, _ = net.SplitHostPort(l.Addr().String())
	}
	// The address clients on this host connect to (VAULT_ADDR, VAULT_AGENT_ADDR).
	clientAddr := net.JoinHostPort(clientHost(host), port)

	server := newServer(cipher)
	go func() {
		if err := server.Serve(l); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	env := map[string]string{
		"VAULT_ADDR":       "http://" + clientAddr,
		"VAULT_AGENT_ADDR": "http://" + clientAddr,
		"VAULT_TOKEN":      "dummy",
	}
	if keyID != "" {
		env["SOPS_VAULT_URIS"] = fmt.Sprintf("http://%s/v1/transit/encrypt/%s", fileAddr, keyID)
	}
	return env, server.Shutdown, nil
}

// clientHost returns the host for clients on this host to connect to the
// server listening on host. An empty or unspecified host (e.g. ":0",
// "0.0.0.0:0", "[::]:0") is replaced with the loopback address.
func clientHost(host string) string {
	if host == "" {
		return "127.0.0.1"
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		if ip.To4() != nil {
			return "127.0.0.1"
		}
		return "::1"
	}
	return host
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

// cipherErrorStatus returns the HTTP status code for an error returned by Cipher.
// It returns the status code of StatusError if present, otherwise 500.
func cipherErrorStatus(err error) int {
	if e, ok := errors.AsType[*StatusError](err); ok {
		return e.StatusCode
	}
	return http.StatusInternalServerError
}

// EncryptHandlerFunc returns an HTTP handler for Vault Transit Engine encrypt endpoint.
func EncryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue(KeyIDPathParam)
		slog.Debug("Encrypting data with Sakura KMS", "key_id", keyID)
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
			errorResponse(w, err, cipherErrorStatus(err))
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
		slog.Debug("Decrypting data with Sakura KMS", "key_id", keyID)
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
			errorResponse(w, err, cipherErrorStatus(err))
			return
		}
		// Encode plaintext as base64 for response
		res := &VaultDecryptResponse{
			Plaintext: base64.StdEncoding.EncodeToString(plaintext),
		}
		jsonResponse(w, http.StatusOK, res)
	}
}
