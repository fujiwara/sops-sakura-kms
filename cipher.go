package ssk

import (
	"context"
	"fmt"

	"github.com/sacloud/kms-api-go"
	v1 "github.com/sacloud/kms-api-go/apis/v1"
)

type Cipher interface {
	Encrypt(ctx context.Context, keyID string, plaintext []byte) (string, error)
	Decrypt(ctx context.Context, keyID string, ciphertext string) ([]byte, error)
}

type SakuraKMS struct {
	client *v1.Client
}

func NewSakuraKMS() (*SakuraKMS, error) {
	client, err := kms.NewClient()
	if err != nil {
		return nil, err
	}
	return &SakuraKMS{
		client: client,
	}, nil
}

func (c *SakuraKMS) Encrypt(ctx context.Context, keyID string, plaintext []byte) (string, error) {
	keyOp := kms.NewKeyOp(c.client)
	ciphertext, err := keyOp.Encrypt(ctx, keyID, plaintext, v1.KeyEncryptAlgoEnumAes256Gcm)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}
	return ciphertext, nil
}

func (c *SakuraKMS) Decrypt(ctx context.Context, keyID string, ciphertext string) ([]byte, error) {
	keyOp := kms.NewKeyOp(c.client)
	plaintext, err := keyOp.Decrypt(ctx, keyID, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}
