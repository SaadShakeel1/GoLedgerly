package auth

import (
	"testing"
	"time"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	secret := []byte("s3cr3t")
	token, err := Sign(secret, "alice", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(secret, token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.Subject != "alice" {
		t.Errorf("subject: got %q, want %q", claims.Subject, "alice")
	}
}

func TestVerify_Expired(t *testing.T) {
	secret := []byte("s3cr3t")
	token, _ := Sign(secret, "alice", -time.Second) // already expired
	if _, err := Verify(secret, token); err != ErrExpiredToken {
		t.Errorf("got %v, want ErrExpiredToken", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	token, _ := Sign([]byte("right"), "alice", time.Minute)
	if _, err := Verify([]byte("wrong"), token); err != ErrBadSignature {
		t.Errorf("got %v, want ErrBadSignature", err)
	}
}

func TestVerify_Tampered(t *testing.T) {
	secret := []byte("s3cr3t")
	token, _ := Sign(secret, "alice", time.Minute)
	tampered := token[:len(token)-3] + "aaa" // mangle the signature
	if _, err := Verify(secret, tampered); err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}
