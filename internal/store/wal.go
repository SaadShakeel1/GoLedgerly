package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	walVersion       = 1
	maxWALRecordSize = 16 << 20

	walOperationSet    = "set"
	walOperationDelete = "delete"
)

// ErrWALRecordTooLarge is returned before a mutation is logged or applied.
var ErrWALRecordTooLarge = errors.New("wal record exceeds maximum size")

type walRecord struct {
	Version    int     `json:"version"`
	Sequence   uint64  `json:"sequence"`
	Operation  string  `json:"operation"`
	Key        string  `json:"key"`
	Value      string  `json:"value,omitempty"`
	ExpiresAt  int64   `json:"expires_at,omitempty"`
	EvictedKey *string `json:"evicted_key,omitempty"`
}

type WAL struct {
	mu           sync.Mutex
	path         string
	file         *os.File
	writer       *bufio.Writer
	lastSequence uint64
}

func NewWAL(path string) (*WAL, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, err
	}

	return &WAL{
		path:   path,
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (w *WAL) LogSet(
	key string,
	value string,
	expiry time.Time,
	evictedKey *string,
) error {
	record := walRecord{
		Version:    walVersion,
		Operation:  walOperationSet,
		Key:        key,
		Value:      value,
		EvictedKey: evictedKey,
	}
	if !expiry.IsZero() {
		record.ExpiresAt = expiry.UnixMilli()
	}

	return w.appendRecord(record)
}

func (w *WAL) LogDelete(key string) error {
	return w.appendRecord(walRecord{
		Version:   walVersion,
		Operation: walOperationDelete,
		Key:       key,
	})
}

func (w *WAL) appendRecord(record walRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	record.Sequence = w.lastSequence + 1
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode WAL record: %w", err)
	}
	if len(encoded)+1 > maxWALRecordSize {
		return ErrWALRecordTooLarge
	}

	encoded = append(encoded, '\n')
	if _, err := w.writer.Write(encoded); err != nil {
		return fmt.Errorf("write WAL record: %w", err)
	}
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("flush WAL record: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync WAL record: %w", err)
	}

	w.lastSequence = record.Sequence
	return nil
}

func (w *WAL) setLastSequence(sequence uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.lastSequence = sequence
}

func (w *WAL) truncate(size int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("flush WAL before truncation: %w", err)
	}
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close WAL before truncation: %w", err)
	}
	if err := os.Truncate(w.path, size); err != nil {
		return fmt.Errorf("truncate incomplete WAL record: %w", err)
	}

	file, err := os.OpenFile(
		w.path,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("reopen WAL after truncation: %w", err)
	}
	w.file = file
	w.writer = bufio.NewWriter(file)

	return nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return err
	}

	return w.file.Close()
}

func ReplayWAL(walPath string, store *Store) error {
	file, err := os.Open(walPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	now := time.Now()
	var validSize int64
	var lastSequence uint64
	lineNumber := 0

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxWALRecordSize)
	scanner.Split(splitCompleteWALLines)

	for scanner.Scan() {
		rawLine := scanner.Bytes()
		validSize += int64(len(rawLine) + 1)
		lineNumber++

		line := bytes.TrimSuffix(rawLine, []byte{'\r'})
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		if bytes.HasPrefix(bytes.TrimSpace(line), []byte{'{'}) {
			record, err := decodeWALRecord(line)
			if err != nil {
				return fmt.Errorf("replay WAL line %d: %w", lineNumber, err)
			}
			if record.Sequence != lastSequence+1 {
				return fmt.Errorf(
					"replay WAL line %d: sequence %d follows %d",
					lineNumber,
					record.Sequence,
					lastSequence,
				)
			}

			if err := applyWALRecord(store, record, now); err != nil {
				return fmt.Errorf("replay WAL line %d: %w", lineNumber, err)
			}
			lastSequence = record.Sequence
			continue
		}

		if err := applyLegacyWALRecord(store, string(line), now); err != nil {
			return fmt.Errorf("replay legacy WAL line %d: %w", lineNumber, err)
		}
		lastSequence++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan WAL: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat WAL: %w", err)
	}
	if info.Size() > validSize {
		if err := file.Close(); err != nil {
			return fmt.Errorf("close WAL before truncation: %w", err)
		}
		if err := store.wal.truncate(validSize); err != nil {
			return err
		}
	}

	store.wal.setLastSequence(lastSequence)
	return nil
}

func splitCompleteWALLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if index := bytes.IndexByte(data, '\n'); index >= 0 {
		return index + 1, data[:index], nil
	}
	if atEOF {
		return len(data), nil, nil
	}
	return 0, nil, nil
}

func decodeWALRecord(line []byte) (walRecord, error) {
	var record walRecord
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return walRecord{}, fmt.Errorf("decode structured record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return walRecord{}, errors.New("structured record contains trailing data")
	}
	if record.Version != walVersion {
		return walRecord{}, fmt.Errorf("unsupported WAL version %d", record.Version)
	}
	if record.Sequence == 0 {
		return walRecord{}, errors.New("structured record has zero sequence")
	}
	if record.Operation != walOperationSet && record.Operation != walOperationDelete {
		return walRecord{}, fmt.Errorf("unsupported WAL operation %q", record.Operation)
	}

	return record, nil
}

func applyWALRecord(store *Store, record walRecord, now time.Time) error {
	switch record.Operation {
	case walOperationSet:
		var expiry time.Time
		if record.ExpiresAt != 0 {
			expiry = time.UnixMilli(record.ExpiresAt)
		}
		store.applySetFromWAL(record.Key, record.Value, expiry, record.EvictedKey, now)
		return nil

	case walOperationDelete:
		store.del(record.Key)
		return nil

	default:
		return fmt.Errorf("unsupported WAL operation %q", record.Operation)
	}
}

func applyLegacyWALRecord(store *Store, line string, now time.Time) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "SET":
		if len(parts) < 3 {
			return errors.New("invalid legacy SET record")
		}
		store.set(parts[1], strings.Join(parts[2:], " "), time.Time{})
		return nil

	case "SET_TTL":
		if len(parts) < 4 {
			return errors.New("invalid legacy SET_TTL record")
		}
		expiryMilliseconds, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse legacy expiry: %w", err)
		}

		key := parts[1]
		expiry := time.UnixMilli(expiryMilliseconds)
		if !now.Before(expiry) {
			store.del(key)
			return nil
		}
		store.set(key, strings.Join(parts[2:len(parts)-1], " "), expiry)
		return nil

	case "DEL":
		if len(parts) != 2 {
			return errors.New("invalid legacy DEL record")
		}
		store.del(parts[1])
		return nil

	default:
		return fmt.Errorf("unknown legacy operation %q", parts[0])
	}
}
