package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Claims is the JSON payload carried inside the token.
type Claims struct {
	Subject string `json:"sub"` // who the token is for
	Expiry  int64  `json:"exp"` // unix seconds; when it stops being valid
}

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

var (
	ErrMalformedToken = errors.New("auth: malformed token")
	ErrBadSignature   = errors.New("auth: invalid signature")
	ErrExpiredToken   = errors.New("auth: token expired")
)

// Sign builds a signed JWT: base64(header).base64(claims).base64(signature)
func Sign(secret []byte, subject string, ttl time.Duration) (string, error) {
	hJSON, err := json.Marshal(header{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	cJSON, err := json.Marshal(Claims{
		Subject: subject,
		Expiry:  time.Now().Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}

	signingInput := b64(hJSON) + "." + b64(cJSON)
	return signingInput + "." + sign(secret, signingInput), nil
}

// Verify checks the signature and expiry, returning the claims if valid.
func Verify(secret []byte, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := sign(secret, signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrBadSignature
	}

	cJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformedToken
	}
	var claims Claims
	if err := json.Unmarshal(cJSON, &claims); err != nil {
		return nil, ErrMalformedToken
	}

	if time.Now().Unix() >= claims.Expiry {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func sign(secret []byte, input string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return b64(mac.Sum(nil))
}
