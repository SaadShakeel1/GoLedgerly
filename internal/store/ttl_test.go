package store

import (
	"os"
	"testing"
	"time"
)

func TestTTL_LazyExpiration(t *testing.T) {
	walFile := "test_ttl_lazy.wal"
	defer os.Remove(walFile)
	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
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
	walFile := "test_ttl_active.wal"
	defer os.Remove(walFile)
	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	s.SetWithTTL("k", "v", 50*time.Millisecond)

	time.Sleep(125 * time.Millisecond)

	_, ok := s.Get("k")
	if ok {
		t.Fatal("expected key to be removed by active sweeper")
	}
}
func TestLRU_Eviction(t *testing.T) {
	walFile := "test_lru.wal"
	defer os.Remove(walFile)
	s, err := New(3, walFile)
	if err != nil {
		t.Fatal(err)
	}
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
func TestWAL_Replay(t *testing.T) {
	walFile := "test_replay.wal"
	defer os.Remove(walFile)

	s1, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	s1.Set("user:1", "Alice")
	s1.Set("user:2", "Bob")
	s1.SetWithTTL("session:1", "active", 1*time.Hour)
	s1.Stop()

	s2, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Stop()

	v1, ok1 := s2.Get("user:1")
	if !ok1 || v1 != "Alice" {
		t.Fatal("expected 'user:1' to be replayed")
	}
	v2, ok2 := s2.Get("user:2")
	if !ok2 || v2 != "Bob" {
		t.Fatal("expected 'user:2' to be replayed")
	}
	v3, ok3 := s2.Get("session:1")
	if !ok3 || v3 != "active" {
		t.Fatal("expected 'session:1' to be replayed")
	}
}
