package store

import (
	"testing"
	"time"
)

func TestTTL_LazyExpiration(t *testing.T) {
	s := New()
	s.SetWithTTL("k", "v", 50*time.Millisecond)

	v, ok := s.Get("k")
	if !ok || v != "v" {
		t.Fatalf("expected %q, got %q ok=%v", "v", v, ok)
	}

	time.Sleep(75 * time.Millisecond)

	_, ok = s.Get("k")
	if ok {
		t.Fatalf("expected key to be expired and removed")
	}
}
