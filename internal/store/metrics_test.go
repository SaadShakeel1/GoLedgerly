package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMetrics_HitsAndMisses(t *testing.T) {
	walFile := filepath.Join(t.TempDir(), "metrics.wal")
	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	s.Set("k", "v") // one key now lives
	s.Get("k")      // hit
	s.Get("k")      // hit
	s.Get("absent") // miss

	stats := s.Stats()
	if stats["hits"] != 2 {
		t.Errorf("hits: got %d, want 2", stats["hits"])
	}
	if stats["misses"] != 1 {
		t.Errorf("misses: got %d, want 1", stats["misses"])
	}
	if stats["keys"] != 1 {
		t.Errorf("keys: got %d, want 1", stats["keys"])
	}
}

func TestMetrics_Evictions(t *testing.T) {
	walFile := filepath.Join(t.TempDir(), "metrics.wal")
	s, err := New(2, walFile) // capacity 2
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	s.Set("a", "1")
	s.Set("b", "2")
	s.Set("c", "3") // exceeds capacity -> evicts the LRU key

	stats := s.Stats()
	if stats["evictions"] != 1 {
		t.Errorf("evictions: got %d, want 1", stats["evictions"])
	}
	if stats["keys"] != 2 {
		t.Errorf("keys: got %d, want 2", stats["keys"])
	}
}

func TestMetrics_Expirations(t *testing.T) {
	walFile := filepath.Join(t.TempDir(), "metrics.wal")
	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	s.SetWithTTL("k", "v", 20*time.Millisecond)

	// Wait past the TTL and past at least one sweeper tick (250ms).
	time.Sleep(300 * time.Millisecond)

	stats := s.Stats()
	if stats["expirations"] != 1 {
		t.Errorf("expirations: got %d, want 1", stats["expirations"])
	}
	if stats["keys"] != 0 {
		t.Errorf("keys: got %d, want 0", stats["keys"])
	}
}
