package store

import (
	"os"
	"testing"
)

func TestStoreSetAndGet(t *testing.T) {
	walFile := "test_store.wal"
	defer os.Remove(walFile)

	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	err = s.Set("hello", "world")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := s.Get("hello")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "world" {
		t.Fatalf("expected %q, got %q", "world", got)
	}
}

func TestStoreDelete(t *testing.T) {
	walFile := "test_delete.wal"
	defer os.Remove(walFile)

	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	s.Set("hello", "world")
	s.Delete("hello")

	_, ok := s.Get("hello")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestStoreGetMissing(t *testing.T) {
	walFile := "test_missing.wal"
	defer os.Remove(walFile)

	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	_, ok := s.Get("missing")
	if ok {
		t.Fatal("expected missing key")
	}
}
