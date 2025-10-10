package ssk_test

import (
	"os"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

func TestEncryptDecrypt(t *testing.T) {
	keyID := os.Getenv("KEY_ID")
	if keyID == "" {
		t.Skip("KEY_ID is not set")
	}
	c, err := ssk.NewSakuraKMS()
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("Hello, World!")
	ciphertext, err := c.Encrypt(t.Context(), keyID, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Ciphertext:", ciphertext)

	decrypted, err := c.Decrypt(t.Context(), keyID, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Decrypted:", string(decrypted))

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text does not match original plaintext: got %q, want %q", decrypted, plaintext)
	}
}
