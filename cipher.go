package ssk

import (
	"context"
	"fmt"

	"github.com/sacloud/kms-api-go"
	v1 "github.com/sacloud/kms-api-go/apis/v1"
)

// Cipher defines the interface for encryption and decryption operations.
type Cipher interface {
	// Encrypt encrypts plaintext using the specified key ID.
	// Returns base64-encoded ciphertext string.
	Encrypt(ctx context.Context, keyID string, plaintext []byte) (string, error)
	// Decrypt decrypts ciphertext using the specified key ID.
	// Accepts base64-encoded ciphertext string and returns plaintext bytes.
	Decrypt(ctx context.Context, keyID string, ciphertext string) ([]byte, error)
}

// SakuraKMS implements Cipher interface using Sakura Cloud KMS.
type SakuraKMS struct {
	client *v1.Client
}

// NewSakuraKMS creates a new SakuraKMS instance.
// It reads credentials from environment variables (SAKURACLOUD_ACCESS_TOKEN, SAKURACLOUD_ACCESS_TOKEN_SECRET).
func NewSakuraKMS() (*SakuraKMS, error) {
	client, err := kms.NewClient()
	if err != nil {
		return nil, err
	}
	return &SakuraKMS{
		client: client,
	}, nil
}

// Encrypt encrypts plaintext using Sakura Cloud KMS with AES-256-GCM algorithm.
func (c *SakuraKMS) Encrypt(ctx context.Context, keyID string, plaintext []byte) (string, error) {
	keyOp := kms.NewKeyOp(c.client)
	ciphertext, err := keyOp.Encrypt(ctx, keyID, plaintext, v1.KeyEncryptAlgoEnumAes256Gcm)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using Sakura Cloud KMS.
func (c *SakuraKMS) Decrypt(ctx context.Context, keyID string, ciphertext string) ([]byte, error) {
	keyOp := kms.NewKeyOp(c.client)
	plaintext, err := keyOp.Decrypt(ctx, keyID, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
