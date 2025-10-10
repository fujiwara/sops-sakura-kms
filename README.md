# sops-sakura-kms

A SOPS wrapper that enables [SOPS (Secrets OPerationS)](https://github.com/getsops/sops) to use [Sakura Cloud KMS](https://cloud.sakura.ad.jp/products/kms/) for data key encryption.

This tool acts as a Vault Transit Engine compatible HTTP server, allowing SOPS to encrypt and decrypt data keys using Sakura Cloud KMS through the `--hc-vault-transit` option.

## Features

- **SOPS Integration**: Seamlessly integrates SOPS with Sakura Cloud KMS
- **Vault Compatibility**: Implements Vault Transit Engine compatible API
- **Automatic Configuration**: Automatically configures SOPS with the correct Vault Transit URI
- **Transparent Operation**: Works as a wrapper around SOPS, passing through all SOPS commands

## Installation

```bash
go install github.com/fujiwara/sops-sakura-kms/cmd/sops-sakura-kms@latest
```

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
export SAKURA_KMS_KEY_ID="123456789012"
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
2. Automatically adds `--hc-vault-transit http://127.0.0.1:8200/v1/transit/encrypt/{key_id}` to SOPS arguments
3. Sets required environment variables (`VAULT_ADDR`, `VAULT_TOKEN`)
4. Executes SOPS with the configured parameters
5. The server handles encryption/decryption requests from SOPS using Sakura Cloud KMS

### SOPS Configuration

You can use the standard SOPS configuration file (`.sops.yaml`) without specifying the `hc_vault_transit_uri`:

```yaml
creation_rules:
  - path_regex: \.yaml$
    # No need to specify hc_vault_transit_uri - it's automatically configured
```

**Note**: Do not manually specify `--hc-vault-transit` when using `sops-sakura-kms`. The wrapper will return an error if you do, as it automatically manages this configuration.

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
