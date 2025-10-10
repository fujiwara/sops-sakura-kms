package ssk

type VaultEncryptRequest struct {
	Plaintext string `json:"plaintext"`
}

type VaultEncryptResponse struct {
	Ciphertext string `json:"ciphertext"`
}

type VaultDecryptRequest struct {
	Ciphertext string `json:"ciphertext"`
}

type VaultDecryptResponse struct {
	Plaintext string `json:"plaintext"`
}
