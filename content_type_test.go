package ssk_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

func TestContentTypeValidation(t *testing.T) {
	cipher := &mockCipher{}
	mux := ssk.NewMux(cipher)

	tests := []struct {
		name        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "valid application/json",
			contentType: "application/json",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "valid application/json with charset",
			contentType: "application/json; charset=utf-8",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "empty content-type (allowed)",
			contentType: "",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "invalid text/plain",
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid application/xml",
			contentType: "application/xml",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"plaintext":"dGVzdA=="}`)
			req := httptest.NewRequest("PUT", "/v1/transit/encrypt/test-key", body)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
