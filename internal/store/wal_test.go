package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestReplayPreservesLRUEviction(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")

	original, err := New(3, walPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []struct {
		key   string
		value string
	}{
		{key: "a", value: "1"},
		{key: "b", value: "2"},
		{key: "c", value: "3"},
	} {
		if err := original.Set(item.key, item.value); err != nil {
			t.Fatal(err)
		}
	}

	if _, found := original.Get("a"); !found {
		t.Fatal("expected a before eviction")
	}

	if err := original.Set("d", "4"); err != nil {
		t.Fatal(err)
	}

	if _, found := original.Get("b"); found {
		t.Fatal("expected b to be evicted before restart")
	}

	if err := original.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(3, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	for key, want := range map[string]string{
		"a": "1",
		"c": "3",
		"d": "4",
	} {
		got, found := restored.Get(key)
		if !found || got != want {
			t.Errorf("restored %q: got %q, found=%v; want %q", key, got, found, want)
		}
	}

	if _, found := restored.Get("b"); found {
		t.Fatal("evicted key b reappeared after replay")
	}
}

func TestReplayPreservesLargeMultilineValue(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")
	key := "key with spaces"
	value := " first line\nsecond\tline  " + strings.Repeat("x", 128*1024)

	original, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := original.Set(key, value); err != nil {
		t.Fatal(err)
	}
	if err := original.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	got, found := restored.Get(key)
	if !found || got != value {
		t.Fatalf("large multiline value changed during replay: found=%v, length=%d", found, len(got))
	}
}

func TestReplayTruncatesIncompleteFinalRecord(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")

	original, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := original.Set("first", "value"); err != nil {
		t.Fatal(err)
	}
	if err := original.Stop(); err != nil {
		t.Fatal(err)
	}

	file, err := os.OpenFile(walPath, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"version":1,"sequence":2`); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.Set("second", "value"); err != nil {
		t.Fatal(err)
	}
	if err := recovered.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	for _, key := range []string{"first", "second"} {
		if _, found := restored.Get(key); !found {
			t.Errorf("expected %q after recovery from truncated record", key)
		}
	}
}

func TestReplayRejectsCorruptCompleteRecord(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")
	if err := os.WriteFile(walPath, []byte("{not-json}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := New(0, walPath)
	if err == nil {
		store.Stop()
		t.Fatal("expected corrupt complete WAL record to fail replay")
	}
}

func TestReplaySupportsLegacyAndStructuredRecords(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")
	if err := os.WriteFile(walPath, []byte("SET legacy value with spaces\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	upgraded, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Set("structured", "new value"); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	for key, want := range map[string]string{
		"legacy":     "value with spaces",
		"structured": "new value",
	} {
		got, found := restored.Get(key)
		if !found || got != want {
			t.Errorf("restored %q: got %q, found=%v; want %q", key, got, found, want)
		}
	}
}

func TestWALRejectsOversizedRecordWithoutApplyingIt(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "store.wal")
	store, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Set("oversized", strings.Repeat("x", maxWALRecordSize))
	if !errors.Is(err, ErrWALRecordTooLarge) {
		t.Fatalf("got error %v, want %v", err, ErrWALRecordTooLarge)
	}
	if _, found := store.Get("oversized"); found {
		t.Fatal("oversized value was applied despite WAL rejection")
	}

	if err := store.Set("valid", "value"); err != nil {
		t.Fatal(err)
	}
	if err := store.Stop(); err != nil {
		t.Fatal(err)
	}

	restored, err := New(0, walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()

	if got, found := restored.Get("valid"); !found || got != "value" {
		t.Fatalf("valid write after rejection was not replayed: got %q, found=%v", got, found)
	}
}
