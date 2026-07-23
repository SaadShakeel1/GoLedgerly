package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/SaadShakeel1/goledgerly/internal/auth"
)

// contextKey is a private type so no other package can collide with our key.
type contextKey string

const claimsContextKey = contextKey("claims")

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public routes: health probes and the token-issuing endpoint.
		// (You can't present a token when you're asking for one.)
		if r.URL.Path == "/health" || r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(header, prefix)
		claims, err := auth.Verify(app.jwtSecret, token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Attach the verified claims to the request so downstream handlers
		// know who the caller is without re-parsing the token.
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// claimsFromContext pulls the claims the middleware attached.
func claimsFromContext(r *http.Request) (*auth.Claims, bool) {
	claims, ok := r.Context().Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}
