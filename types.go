package ssk

// VaultEncryptRequest represents the request body for Vault Transit Engine encrypt API.
// Plaintext must be base64-encoded string.
type VaultEncryptRequest struct {
	Plaintext string `json:"plaintext"`
}

// VaultEncryptResponse represents the response body for Vault Transit Engine encrypt API.
// Ciphertext includes "vault:v1:" prefix followed by the encrypted data.
type VaultEncryptResponse struct {
	Ciphertext string `json:"ciphertext"`
}

// VaultDecryptRequest represents the request body for Vault Transit Engine decrypt API.
// Ciphertext must include "vault:v1:" prefix.
type VaultDecryptRequest struct {
	Ciphertext string `json:"ciphertext"`
}

// VaultDecryptResponse represents the response body for Vault Transit Engine decrypt API.
// Plaintext is returned as base64-encoded string.
type VaultDecryptResponse struct {
	Plaintext string `json:"plaintext"`
}

// VaultErrorResponse represents the error response body for Vault API.
// Errors is an array of error message strings.
type VaultErrorResponse struct {
	Errors []string `json:"errors"`
}
