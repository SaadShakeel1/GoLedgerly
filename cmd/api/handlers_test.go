package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SaadShakeel1/goledgerly/internal/auth"
)

func TestRefreshHandler(t *testing.T) {
	secret := []byte("test-jwt-secret")
	app := &application{jwtSecret: secret}

	// Simulate what the middleware does: attach verified claims to the request.
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	ctx := context.WithValue(req.Context(), claimsContextKey, &auth.Claims{Subject: "client"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	app.refreshHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusOK)
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	claims, err := auth.Verify(secret, body.Token)
	if err != nil {
		t.Fatalf("refreshed token did not verify: %v", err)
	}
	if claims.Subject != "client" {
		t.Errorf("subject: got %q, want %q", claims.Subject, "client")
	}
}

func TestRefreshHandler_NoClaims(t *testing.T) {
	app := &application{jwtSecret: []byte("test-jwt-secret")}

	// No claims on the context (as if the route were reached without auth).
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	rr := httptest.NewRecorder()
	app.refreshHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
