package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SaadShakeel1/goledgerly/internal/auth"
)

func TestAuthenticate(t *testing.T) {
	secret := []byte("test-jwt-secret")
	app := &application{jwtSecret: secret}

	validToken, err := auth.Sign(secret, "client", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expiredToken, err := auth.Sign(secret, "client", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}

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
		{"valid token", "/store/k", "Bearer " + validToken, http.StatusOK, true},
		{"expired token", "/store/k", "Bearer " + expiredToken, http.StatusUnauthorized, false},
		{"garbage token", "/store/k", "Bearer not.a.jwt", http.StatusUnauthorized, false},
		{"missing header", "/store/k", "", http.StatusUnauthorized, false},
		{"health is public", "/health", "", http.StatusOK, true},
		{"login is public", "/login", "", http.StatusOK, true},
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
