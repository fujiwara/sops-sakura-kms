package ssk_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

// mockCipher is a mock implementation of Cipher interface for testing
type mockCipher struct{}

func (m *mockCipher) Encrypt(ctx context.Context, keyID string, plaintext []byte) (string, error) {
	// Simply return plaintext as base64-encoded string for testing
	return base64.StdEncoding.EncodeToString(plaintext), nil
}

func (m *mockCipher) Decrypt(ctx context.Context, keyID string, ciphertext string) ([]byte, error) {
	// Simply decode base64 ciphertext for testing
	return base64.StdEncoding.DecodeString(ciphertext)
}

func TestEncryptHandler(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	tests := []struct {
		name           string
		keyID          string
		requestBody    ssk.VaultEncryptRequest
		wantStatus     int
		wantCiphertext string
	}{
		{
			name:  "successful encryption",
			keyID: "test-key-123",
			requestBody: ssk.VaultEncryptRequest{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
			},
			wantStatus:     http.StatusOK,
			wantCiphertext: ssk.VaultPrefix + base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
		},
		{
			name:  "encrypt empty plaintext",
			keyID: "test-key-456",
			requestBody: ssk.VaultEncryptRequest{
				Plaintext: base64.StdEncoding.EncodeToString([]byte("")),
			},
			wantStatus:     http.StatusOK,
			wantCiphertext: ssk.VaultPrefix + base64.StdEncoding.EncodeToString([]byte("")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("PUT", "/v1/transit/encrypt/"+tt.keyID, bytes.NewReader(body))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var res ssk.VaultEncryptResponse
				if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
					t.Fatal(err)
				}
				if res.Ciphertext != tt.wantCiphertext {
					t.Errorf("ciphertext = %q, want %q", res.Ciphertext, tt.wantCiphertext)
				}
			}
		})
	}
}

func TestEncryptHandlerInvalidRequest(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	req := httptest.NewRequest("PUT", "/v1/transit/encrypt/test-key", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecryptHandler(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	plaintext1 := "Hello, World!"
	plaintext2 := ""
	ciphertext1 := base64.StdEncoding.EncodeToString([]byte(plaintext1))
	ciphertext2 := base64.StdEncoding.EncodeToString([]byte(plaintext2))

	tests := []struct {
		name          string
		keyID         string
		requestBody   ssk.VaultDecryptRequest
		wantStatus    int
		wantPlaintext string
	}{
		{
			name:  "successful decryption",
			keyID: "test-key-123",
			requestBody: ssk.VaultDecryptRequest{
				Ciphertext: ssk.VaultPrefix + ciphertext1,
			},
			wantStatus:    http.StatusOK,
			wantPlaintext: base64.StdEncoding.EncodeToString([]byte(plaintext1)),
		},
		{
			name:  "decrypt empty ciphertext",
			keyID: "test-key-456",
			requestBody: ssk.VaultDecryptRequest{
				Ciphertext: ssk.VaultPrefix + ciphertext2,
			},
			wantStatus:    http.StatusOK,
			wantPlaintext: base64.StdEncoding.EncodeToString([]byte(plaintext2)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("PUT", "/v1/transit/decrypt/"+tt.keyID, bytes.NewReader(body))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var res ssk.VaultDecryptResponse
				if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
					t.Fatal(err)
				}
				if res.Plaintext != tt.wantPlaintext {
					t.Errorf("plaintext = %q, want %q", res.Plaintext, tt.wantPlaintext)
				}
			}
		})
	}
}

func TestDecryptHandlerInvalidCiphertext(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	tests := []struct {
		name        string
		requestBody ssk.VaultDecryptRequest
		wantStatus  int
	}{
		{
			name: "missing vault prefix",
			requestBody: ssk.VaultDecryptRequest{
				Ciphertext: "invalid-format",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid json",
			requestBody: ssk.VaultDecryptRequest{
				Ciphertext: "",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("PUT", "/v1/transit/decrypt/test-key", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestDecryptHandlerInvalidRequest(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	req := httptest.NewRequest("PUT", "/v1/transit/decrypt/test-key", bytes.NewReader([]byte("invalid json")))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
