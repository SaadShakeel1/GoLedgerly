package store

import (
	"testing"
	"time"
)

func TestTTL_LazyExpiration(t *testing.T) {
	s := New(0)
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

func TestTTL_ActiveSweeper(t *testing.T) {
	s := New(0)
	defer s.Stop()

	s.SetWithTTL("k", "v", 50*time.Millisecond)

	time.Sleep(125 * time.Millisecond)

	_, ok := s.Get("k")
	if ok {
		t.Fatal("expected key to be removed by active sweeper")
	}
}
func TestLRU_Eviction(t *testing.T) {
	s := New(3)
	defer s.Stop()

	s.Set("a", "1")
	s.Set("b", "2")
	s.Set("c", "3")

	if v, ok := s.Get("a"); !ok || v != "1" {
		t.Fatal("expected 'a' to exist")
	}
	if _, ok := s.Get("b"); !ok {
		t.Fatal("expected 'b' to exist")
	}
	if _, ok := s.Get("c"); !ok {
		t.Fatal("expected 'c' to exist")
	}

	s.Get("a")

	s.Set("d", "4")

	if _, ok := s.Get("a"); !ok {
		t.Fatal("expected 'a' to exist after eviction")
	}
	if _, ok := s.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted (least recently used)")
	}
	if _, ok := s.Get("c"); !ok {
		t.Fatal("expected 'c' to exist after eviction")
	}
	if _, ok := s.Get("d"); !ok {
		t.Fatal("expected 'd' to exist (just added)")
	}
}
