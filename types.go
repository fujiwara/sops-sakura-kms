package ssk

type VaultEncryptRequest struct {
	Plaintext []byte `json:"plaintext"`
}

type VaultEncryptResponse struct {
	Ciphertext string `json:"ciphertext"`
}

type VaultDecryptRequest struct {
	Ciphertext string `json:"ciphertext"`
}

type VaultDecryptResponse struct {
	Plaintext []byte `json:"plaintext"`
}
