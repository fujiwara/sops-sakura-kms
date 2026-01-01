// Package ssk provides a Vault Transit Engine compatible API server
// that enables SOPS to use Sakura Cloud KMS for data key encryption.
//
// # Wrapper Mode
//
// The primary use case is as a SOPS wrapper via the command-line tool.
// See the cmd/sops-sakura-kms package for the CLI entrypoint.
//
// # Library Usage
//
// You can also use this package as a Go library to embed Sakura Cloud KMS-based
// SOPS decryption in your applications. Use [RunServer] to start the
// Vault Transit Engine compatible server, then use the SOPS decrypt package
// to decrypt files.
//
//	addEnv, shutdown, err := ssk.RunServer(ctx, "127.0.0.1:8200", keyID)
//	if err != nil {
//	    return err
//	}
//	defer shutdown(context.Background())
//
//	for k, v := range addEnv {
//	    os.Setenv(k, v)
//	}
//
//	plaintext, err := decrypt.File("secrets.enc.yaml", "yaml")
//
// # Environment Variables
//
// The following environment variables must be set:
//   - SAKURACLOUD_ACCESS_TOKEN: Sakura Cloud API access token
//   - SAKURACLOUD_ACCESS_TOKEN_SECRET: Sakura Cloud API access token secret
//
// For wrapper mode, also set:
//   - SAKURACLOUD_KMS_KEY_ID: Sakura Cloud KMS resource ID (12-digit number)
package ssk
