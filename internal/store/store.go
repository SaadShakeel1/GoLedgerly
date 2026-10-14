package store

import (
	"container/list"
	"sync"
	"time"
)

type entry struct {
	value   string
	expiry  time.Time
	element *list.Element
}

type Store struct {
	mu         sync.RWMutex
	mutationMu sync.Mutex
	data       map[string]entry
	lruList    *list.List
	capacity   int
	stop       chan struct{}
	wal        *WAL
	metrics    *Metrics
}

func New(capacity int, walPath string) (*Store, error) {
	wal, err := NewWAL(walPath)
	if err != nil {
		return nil, err
	}

	s := &Store{
		data:     make(map[string]entry),
		capacity: capacity,
		lruList:  list.New(),
		stop:     make(chan struct{}),
		wal:      wal,
		metrics:  &Metrics{},
	}

	if err := ReplayWAL(walPath, s); err != nil {
		_ = wal.Close()
		return nil, err
	}

	go s.startExpiryLoop(250 * time.Millisecond)

	return s, nil
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.data)
}

func (s *Store) Stats() map[string]int64 {
	return map[string]int64{
		"hits":        s.metrics.Hits.Load(),
		"misses":      s.metrics.Misses.Load(),
		"evictions":   s.metrics.Evictions.Load(),
		"expirations": s.metrics.Expirations.Load(),
		"keys":        int64(s.Len()),
	}
}

func (s *Store) startExpiryLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.deleteExpiredKeys()
		case <-s.stop:
			return
		}
	}
}

func (s *Store) deleteExpiredKeys() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, e := range s.data {
		if !e.expiry.IsZero() && now.After(e.expiry) {
			s.lruList.Remove(e.element)
			delete(s.data, key)
			s.metrics.Expirations.Add(1)
		}
	}
}

func (s *Store) Stop() error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	close(s.stop)
	return s.wal.Close()
}

func (s *Store) Set(key, value string) error {
	return s.setAndLog(key, value, time.Time{})
}

func (s *Store) setAndLog(key, value string, expiry time.Time) error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	evictedKey := s.evictionCandidateLocked(key)
	if err := s.wal.LogSet(key, value, expiry, evictedKey); err != nil {
		return err
	}

	s.applySetLocked(key, value, expiry, evictedKey, true)

	return nil
}

func (s *Store) set(key, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	evictedKey := s.evictionCandidateLocked(key)
	s.applySetLocked(key, value, expiry, evictedKey, false)
}

func (s *Store) evictionCandidateLocked(key string) *string {
	if s.capacity <= 0 || len(s.data) < s.capacity {
		return nil
	}
	if _, exists := s.data[key]; exists {
		return nil
	}

	oldest := s.lruList.Back()
	if oldest == nil {
		return nil
	}

	evictedKey := oldest.Value.(string)
	return &evictedKey
}

func (s *Store) applySetLocked(
	key string,
	value string,
	expiry time.Time,
	evictedKey *string,
	countEviction bool,
) {
	if evictedKey != nil {
		if s.deleteLocked(*evictedKey) && countEviction {
			s.metrics.Evictions.Add(1)
		}
	}

	if existing, ok := s.data[key]; ok {
		s.lruList.Remove(existing.element)
	}

	element := s.lruList.PushFront(key)
	s.data[key] = entry{
		value:   value,
		expiry:  expiry,
		element: element,
	}
}

func (s *Store) applySetFromWAL(
	key string,
	value string,
	expiry time.Time,
	evictedKey *string,
	now time.Time,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if evictedKey != nil {
		s.deleteLocked(*evictedKey)
	}
	if !expiry.IsZero() && !now.Before(expiry) {
		s.deleteLocked(key)
		return
	}

	s.applySetLocked(key, value, expiry, nil, false)
}

func (s *Store) SetWithTTL(key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Set(key, value)
	}

	expiry := time.Now().Add(ttl)
	return s.setAndLog(key, value, expiry)
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]

	if !ok {
		s.mu.RUnlock()
		s.metrics.Misses.Add(1)
		return "", false
	}

	if !e.expiry.IsZero() && time.Now().After(e.expiry) {
		s.mu.RUnlock()

		s.mu.Lock()
		defer s.mu.Unlock()

		currentEntry, stillExists := s.data[key]
		if !stillExists {
			s.metrics.Misses.Add(1)
			return "", false
		}

		if !currentEntry.expiry.IsZero() && time.Now().After(currentEntry.expiry) {
			s.lruList.Remove(currentEntry.element)
			delete(s.data, key)
			s.metrics.Expirations.Add(1)
			s.metrics.Misses.Add(1)
			return "", false
		}

		s.metrics.Hits.Add(1)
		return currentEntry.value, true
	}

	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	currentEntry, stillExists := s.data[key]
	if !stillExists {
		s.metrics.Misses.Add(1)
		return "", false
	}

	s.lruList.MoveToFront(currentEntry.element)
	s.metrics.Hits.Add(1)

	return currentEntry.value, true
}

func (s *Store) Delete(key string) error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.wal.LogDelete(key); err != nil {
		return err
	}

	s.deleteLocked(key)

	return nil
}

func (s *Store) del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteLocked(key)
}

func (s *Store) deleteLocked(key string) bool {
	if existing, ok := s.data[key]; ok {
		s.lruList.Remove(existing.element)
		delete(s.data, key)
		return true
	}

	return false
}
