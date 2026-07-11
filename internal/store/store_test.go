package store

import "testing"

func TestStoreSetAndGet(t *testing.T) {
	s := New(0)

	s.Set("hello", "world")

	got, ok := s.Get("hello")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "world" {
		t.Fatalf("expected %q, got %q", "world", got)
	}
}

func TestStoreDelete(t *testing.T) {
	s := New(0)

	s.Set("hello", "world")
	s.Delete("hello")

	_, ok := s.Get("hello")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestStoreGetMissing(t *testing.T) {
	s := New(0)

	_, ok := s.Get("missing")
	if ok {
		t.Fatal("expected missing key")
	}
}
