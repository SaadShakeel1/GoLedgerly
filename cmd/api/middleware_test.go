package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	app := &application{apiKey: "secret"}

	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	handler := app.authenticate(next)

	tests := []struct {
		name        string
		path        string
		authHeader  string
		wantStatus  int
		wantReached bool
	}{
		{"valid key", "/store/k", "Bearer secret", http.StatusOK, true},
		{"wrong key", "/store/k", "Bearer nope", http.StatusUnauthorized, false},
		{"missing header", "/store/k", "", http.StatusUnauthorized, false},
		{"health is public", "/health", "", http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached = false
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}
			if reached != tt.wantReached {
				t.Errorf("next reached: got %v, want %v", reached, tt.wantReached)
			}
		})
	}
}
