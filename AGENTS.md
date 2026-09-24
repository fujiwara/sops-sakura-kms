# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

sops-sakura-kms is a SOPS wrapper that enables SOPS to use Sakura Cloud KMS for data key encryption. It acts as a Vault Transit Engine compatible HTTP server, allowing SOPS to encrypt and decrypt data keys using Sakura Cloud KMS through the `SOPS_VAULT_URIS` environment variable.

The tool automatically:
1. Starts a local Vault Transit Engine compatible HTTP server
2. Configures SOPS with the correct `SOPS_VAULT_URIS` environment variable
3. Executes SOPS with proper environment variables
4. Handles encryption/decryption requests from SOPS using Sakura Cloud KMS

## Architecture

### Wrapper Mode (Primary Use Case)
The tool operates as a SOPS wrapper via `RunWrapper()` function:
- Reads `SAKURA_KMS_KEY_ID` environment variable (12-digit Sakura Cloud resource ID as string)
- Starts HTTP server on `127.0.0.1:8200` in background
- Waits for server health check (30 retries × 100ms)
- Automatically sets `SOPS_VAULT_URIS=http://127.0.0.1:8200/v1/transit/encrypt/{key_id}` environment variable
- Sets `VAULT_ADDR` and `VAULT_TOKEN` environment variables
- Executes SOPS command with all original arguments

### API Endpoints
- `GET /health` - Health check endpoint (returns 200 OK)
- `PUT /v1/transit/encrypt/{key_id}` - Encrypts plaintext using Sakura Cloud KMS
- `PUT /v1/transit/decrypt/{key_id}` - Decrypts ciphertext using Sakura Cloud KMS

### Key Components
- **main.go**: HTTP server setup, handlers, and `RunWrapper()` implementation
  - `NewMux(cipher Cipher)`: Creates HTTP ServeMux with all endpoints
  - `RunWrapper(ctx, sopsArgs)`: Main wrapper function
  - `EncryptHandlerFunc(cipher)`: Encrypt endpoint handler
  - `DecryptHandlerFunc(cipher)`: Decrypt endpoint handler
  - `waitForServer()`: Polls health endpoint until ready
- **cipher.go**: `Cipher` interface and `SakuraKMS` implementation
  - Uses `github.com/sacloud/sacloud-sdk-go/api/kms` for Sakura Cloud KMS API
  - Encrypts with AES-256-GCM algorithm
- **types.go**: Vault Transit Engine compatible request/response types
  - All plaintext/ciphertext are base64-encoded strings for Vault API compatibility
- **cmd/sops-sakura-kms/main.go**: CLI entrypoint that calls `RunWrapper()`

### Data Format
- **Ciphertext prefix**: All encrypted data includes `vault:v1:` prefix for SOPS/Vault compatibility
- **Base64 encoding**: Plaintext and ciphertext are base64-encoded in API requests/responses
- **Key ID format**: Sakura Cloud KMS resource ID is a 12-digit number (treated as string)

## Development Commands

### Build
```bash
make                    # Build binary to ./sops-sakura-kms
go build -o sops-sakura-kms ./cmd/sops-sakura-kms
```

### Test
```bash
make test              # Run all tests
go test -v ./...       # Run all tests with verbose output
go test -race ./...    # Run tests with race detector (used in CI)
```

Integration tests run against [sakumock](https://github.com/sacloud/sakumock) (in-process Sakura Cloud KMS mock) by default. Set `KEY_ID` environment variable to also run the test that actually calls Sakura Cloud KMS API.

### Install
```bash
make install           # Install to $GOPATH/bin
```

### Release Build
```bash
make dist              # Build release binaries with goreleaser (snapshot mode)
goreleaser build --snapshot --clean
```

## Testing Notes

### Test Structure
- **handler_test.go**: HTTP handler tests using mock cipher
  - Uses `mockCipher` that base64-encodes for encrypt, base64-decodes for decrypt
  - Tests all endpoints including error cases
  - Uses `NewMux()` to ensure consistency with production routing
- **vault_compat_test.go**: Vault API client compatibility tests
  - Uses `github.com/hashicorp/vault/api` client to verify compatibility
  - Tests base64 encoding/decoding flow matches Vault Transit Engine behavior
- **wrapper_test.go**: `RunWrapper()` function tests
  - Tests environment variable validation
  - Does not test full SOPS integration (requires SOPS binary)
- **sakumock_test.go**: Integration tests using `github.com/sacloud/sakumock/kms`
  - Starts an in-process mock KMS server with a fixed key ID (`123456789012`)
  - Tests `SakuraKMS` encrypt/decrypt, `RunServer()` via the Vault API client, and
    end-to-end `RunWrapper()` with the real `sops` binary (skipped if `sops` is not in `PATH`)
  - No credentials required; `SAKURA_ENDPOINTS_KMS` points sacloud-sdk-go at the mock
- **kms_test.go**: Integration tests with actual Sakura Cloud KMS
  - Requires `KEY_ID` environment variable (12-digit resource ID)
  - Skipped if `KEY_ID` is not set
  - Tests real encryption/decryption with Sakura Cloud KMS

### Key Testing Principles
- Mock cipher for unit tests (no external dependencies)
- sakumock for integration tests (no network access or credentials)
- All tests use `NewMux()` to create handlers consistently
- Integration tests can be skipped without blocking development
- Error handling is thoroughly tested (invalid JSON, missing prefix, etc.)

## Important Design Decisions

### Why Wrapper Mode?
The wrapper approach allows dynamic configuration based on `SAKURA_KMS_KEY_ID` by automatically setting the `SOPS_VAULT_URIS` environment variable.

### Why Use SOPS_VAULT_URIS?
SOPS supports the `SOPS_VAULT_URIS` environment variable to configure Vault Transit Engine URIs. Using this environment variable instead of command-line flags prevents issues with argument ordering when executing SOPS.

### Error Handling Best Practices
- 4xx errors from Sakura Cloud KMS API (e.g. 401 without credentials) are returned to the client with the same status code via `StatusError`; other cipher errors are returned as 500. Vault API clients (SOPS) retry on 5xx, so non-retryable errors must not be 500
- JSON encoding errors in response handlers are logged but not fatal (connection may be closed)
- Server startup errors are captured via channel to detect port conflicts
- `http.ErrServerClosed` is ignored (normal shutdown)
- Health check retries 30 times with 100ms intervals (max 3 seconds)

### Constants
- `VaultPrefix = "vault:v1:"` - Required by SOPS for Vault Transit Engine compatibility
- `KeyIDPathParam = "key_id"` - URL path parameter name
- `SAKURA_KMS_KEY_ID` - Environment variable for KMS resource ID (defined by the `env` struct tag in `env.go`; the legacy `SAKURACLOUD_KMS_KEY_ID` is still accepted as a fallback)
- `ServerAddr = "127.0.0.1:8200"` - Standard Vault server address (localhost only)

## Go Version

This project uses Go 1.26+ (as specified in go.mod, required by sacloud-sdk-go). CI tests against Go 1.26 and 1.27.
