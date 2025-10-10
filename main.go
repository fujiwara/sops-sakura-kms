package ssk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

const (
	VaultPrefix    = "vault:v1:"
	KeyIDPathParam = "key_id"
)

func Run(ctx context.Context) error {
	cipher, err := NewSakuraKMS()
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(makeURL("PUT /v1/transit/encrypt/{%s}"), encryptHandlerFunc(cipher))
	mux.HandleFunc(makeURL("PUT /v1/transit/decrypt/{%s}"), decryptHandlerFunc(cipher))
	sv := &http.Server{
		Addr:    "127.0.0.1:8200",
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		slog.Info("shutting down server...")
		sv.Shutdown(context.Background())
	}()
	slog.Info("starting server...", "addr", sv.Addr)
	return sv.ListenAndServe()
}

func makeURL(path string) string {
	return fmt.Sprintf(path, KeyIDPathParam)
}

func errorResponse(w http.ResponseWriter, err error, status int) {
	slog.Error("error response", "status", status, "error", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{
			{
				"status": status,
				"detail": err.Error(),
			},
		},
	})
}

func encryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue("key_id")
		slog.Info("encrypt", "key_id", keyID)
		req := &VaultEncryptRequest{}
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			errorResponse(w, err, http.StatusBadRequest)
			return
		}
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		ciphertext, err := cipher.Encrypt(r.Context(), keyID, req.Plaintext)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		res := &VaultEncryptResponse{
			Ciphertext: VaultPrefix + ciphertext,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

func decryptHandlerFunc(cipher Cipher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		keyID := r.PathValue("key_id")
		slog.Info("decrypt", "key_id", keyID)
		req := &VaultDecryptRequest{}
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			errorResponse(w, err, http.StatusBadRequest)
			return
		}
		body := strings.TrimPrefix(req.Ciphertext, VaultPrefix)
		if len(body) == len(req.Ciphertext) {
			errorResponse(w, fmt.Errorf("invalid ciphertext format"), http.StatusBadRequest)
			return
		}
		plaintext, err := cipher.Decrypt(r.Context(), keyID, body)
		if err != nil {
			errorResponse(w, err, http.StatusInternalServerError)
			return
		}
		res := &VaultDecryptResponse{
			Plaintext: plaintext,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}
