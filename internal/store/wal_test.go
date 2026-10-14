package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReplayExpiredTTLDoesNotResurrectPreviousValue(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")

	original, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := original.Set("key", "old"); err != nil {
		t.Fatal(err)
	}

	if err := original.SetWithTTL(
		"key",
		"temporary",
		10*time.Millisecond,
	); err != nil {
		t.Fatal(err)
	}

	if err := original.Stop(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond)

	restored, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	value, found := restored.Get("key")
	if found {
		t.Fatalf(
			"expired overwrite resurrected previous value %q",
			value,
		)
	}
}
