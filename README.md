# sops-sakura-kms

A SOPS wrapper that enables [SOPS (Secrets OPerationS)](https://github.com/getsops/sops) to use [Sakura Cloud KMS](https://cloud.sakura.ad.jp/products/kms/) for data key encryption.

This tool acts as a Vault Transit Engine compatible HTTP server, allowing SOPS to encrypt and decrypt data keys using Sakura Cloud KMS through the `SOPS_VAULT_URIS` environment variable.

## Features

- **SOPS Integration**: Seamlessly integrates SOPS with Sakura Cloud KMS
- **Vault Compatibility**: Implements Vault Transit Engine compatible API
- **Automatic Configuration**: Automatically configures SOPS with the correct Vault Transit URI
- **Transparent Operation**: Works as a wrapper around SOPS, passing through all SOPS commands
- **Server-Only Mode**: Can run as a standalone Vault Transit Engine compatible server without SOPS

## Installation

### Homebrew

```bash
brew install fujiwara/tap/sops-sakura-kms
```

### Binary releases

Download the latest binary from the [releases page](https://github.com/fujiwara/sops-sakura-kms/releases).

### From source

```bash
go install github.com/fujiwara/sops-sakura-kms/cmd/sops-sakura-kms@latest
```

### GitHub Action

You can install `sops-sakura-kms` in your GitHub Actions workflows.

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: fujiwara/sops-sakura-kms@main
        with:
          version: 'v0.0.4' # or 'latest'
          # version-file: './.version'
```

This action sets up `sops-sakura-kms` and makes it available in the PATH. The `sops` command needs to be installed separately.

### Container Image

You can use the provided container image: [`ghcr.io/fujiwara/sops-sakura-kms`](https://github.com/fujiwara/sops-sakura-kms/pkgs/container/sops-sakura-kms).

```console
$ docker run --rm \
    -e SAKURACLOUD_ACCESS_TOKEN \
    -e SAKURACLOUD_ACCESS_TOKEN_SECRET \
    -e SAKURACLOUD_KMS_KEY_ID \
    -v $(pwd):/work -w /work \
    ghcr.io/fujiwara/sops-sakura-kms:v0.0.6 \
    -d secrets.enc.yaml
```

This image includes both `sops-sakura-kms` and `sops` commands.

## Prerequisites

- [SOPS](https://github.com/getsops/sops) must be installed and available in your PATH
- Sakura Cloud API credentials must be set in environment variables

## Configuration

Set the following environment variables:

```bash
# Sakura Cloud API credentials
export SAKURACLOUD_ACCESS_TOKEN="your-access-token"
export SAKURACLOUD_ACCESS_TOKEN_SECRET="your-access-token-secret"

# Sakura Cloud KMS Resource ID (12-digit number as string, e.g., 123456789012)
export SAKURACLOUD_KMS_KEY_ID="123456789012"
```

### Optional Environment Variables

You can customize the behavior with these optional environment variables:

```bash
# Run server-only mode without executing SOPS (default: false)
export SSK_SERVER_ONLY=true

# Server listen address (default: 127.0.0.1:8200)
export SSK_SERVER_ADDR="127.0.0.1:8200"

# SOPS command path (default: sops)
export SSK_SOPS_PATH="/path/to/sops"
```

## Usage

Use `sops-sakura-kms` as a drop-in replacement for the `sops` command:

```bash
# Encrypt a file
sops-sakura-kms -e secrets.yaml > secrets.enc.yaml

# Decrypt a file
sops-sakura-kms -d secrets.enc.yaml

# Edit an encrypted file
sops-sakura-kms secrets.enc.yaml
```

### How it works

1. `sops-sakura-kms` starts a local Vault Transit Engine compatible HTTP server on `127.0.0.1:8200`
2. Automatically sets the `SOPS_VAULT_URIS` environment variable to `http://127.0.0.1:8200/v1/transit/encrypt/{key_id}`
3. Sets required environment variables (`VAULT_ADDR`, `VAULT_TOKEN`)
4. Executes SOPS with the configured environment
5. The server handles encryption/decryption requests from SOPS using Sakura Cloud KMS

### SOPS Configuration

You can use the standard SOPS configuration file (`.sops.yaml`) without specifying the `hc_vault_transit_uri`:

```yaml
creation_rules:
  - path_regex: \.yaml$
    # No need to specify hc_vault_transit_uri - it's automatically configured via SOPS_VAULT_URIS
```

**Note**: The wrapper automatically sets the `SOPS_VAULT_URIS` environment variable, so you don't need to configure it manually in `.sops.yaml` or pass it as a command-line argument.

### Server-Only Mode

You can run `sops-sakura-kms` as a standalone Vault Transit Engine compatible server without executing SOPS:

```bash
# Start server-only mode
export SSK_SERVER_ONLY=true
sops-sakura-kms

# The server will run until interrupted (Ctrl+C)
```

This mode is useful when:
- You want to use the Vault Transit Engine API directly from your applications
- You need to run the server as a separate service
- You're testing or debugging the encryption/decryption API endpoints

In server-only mode, you can use the Vault API endpoints directly:

```bash
# Encrypt data
curl -X PUT http://127.0.0.1:8200/v1/transit/encrypt/123456789012 \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"aGVsbG8gd29ybGQ="}'

# Decrypt data
curl -X PUT http://127.0.0.1:8200/v1/transit/decrypt/123456789012 \
  -H "Content-Type: application/json" \
  -d '{"ciphertext":"vault:v1:..."}'
```

## API Endpoints

The tool provides the following Vault Transit Engine compatible endpoints:

- `GET /health` - Health check endpoint
- `PUT /v1/transit/encrypt/{key_id}` - Encrypt data using specified KMS key
- `PUT /v1/transit/decrypt/{key_id}` - Decrypt data using specified KMS key

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with actual Sakura Cloud KMS (requires credentials and KEY_ID)
KEY_ID=123456789012 go test ./...
```

### Building

```bash
make build
```

## License

MIT License - see [LICENSE](LICENSE) file for details

## Author

FUJIWARA Shunichiro
