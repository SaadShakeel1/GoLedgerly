package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

func TestConcurrentWritesSurviveReplay(t *testing.T) {
	walFile := filepath.Join(t.TempDir(), "concurrent.wal")

	s, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}

	const writers = 50

	var wg sync.WaitGroup
	errors := make(chan error, writers)

	for i := 0; i < writers; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", id)
			value := fmt.Sprintf("value-%d", id)

			if err := s.Set(key, value); err != nil {
				errors <- fmt.Errorf("set %q: %w", key, err)
				return
			}

			if err := s.Set("shared", value); err != nil {
				errors <- fmt.Errorf("set shared key: %w", err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Fatal(err)
	}

	sharedBeforeRestart, ok := s.Get("shared")
	if !ok {
		t.Fatal("expected shared key before restart")
	}

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(0, walFile)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	for i := 0; i < writers; i++ {
		key := fmt.Sprintf("key-%d", i)
		want := fmt.Sprintf("value-%d", i)

		got, ok := restored.Get(key)
		if !ok {
			t.Errorf("expected %q to survive replay", key)
			continue
		}

		if got != want {
			t.Errorf("value for %q: got %q, want %q", key, got, want)
		}
	}

	sharedAfterRestart, ok := restored.Get("shared")
	if !ok {
		t.Fatal("expected shared key after restart")
	}

	if sharedAfterRestart != sharedBeforeRestart {
		t.Fatalf(
			"shared value changed after replay: got %q, want %q",
			sharedAfterRestart,
			sharedBeforeRestart,
		)
	}
}
