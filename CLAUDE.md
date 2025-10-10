# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

sops-sakura-kms is a Vault Transit Engine compatible HTTP server that provides encryption/decryption functionality using Sakura Cloud KMS. This allows SOPS (Secrets OPerationS) to use Sakura Cloud KMS as a backend through the Vault Transit Engine interface.

## Architecture

The project implements a minimal HTTP server that exposes two Vault-compatible endpoints:
- `PUT /v1/transit/encrypt/{key_id}` - encrypts plaintext using Sakura Cloud KMS
- `PUT /v1/transit/decrypt/{key_id}` - decrypts ciphertext using Sakura Cloud KMS

Key components:
- **main.go**: HTTP server setup, request handlers, and routing. Listens on `127.0.0.1:8200` and implements Vault Transit Engine compatible API
- **cipher.go**: `Cipher` interface and `SakuraKMS` implementation that wraps the Sakura Cloud KMS client
- **types.go**: Request/response types matching Vault Transit Engine format
- **cmd/sops-sakura-kms/main.go**: CLI entrypoint with signal handling

Ciphertext format: All encrypted data is prefixed with `vault:v1:` to maintain compatibility with SOPS/Vault.

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

Set `KEY_ID` environment variable to run integration tests that actually call Sakura Cloud KMS API.

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

The test suite in kms_test.go requires `KEY_ID` environment variable to be set for integration testing. Tests will be skipped if this variable is not set. When adding new functionality, ensure tests can run both with and without actual KMS credentials.

## Go Version

This project uses Go 1.24+ (as specified in go.mod). CI tests against Go 1.23 and 1.24.
