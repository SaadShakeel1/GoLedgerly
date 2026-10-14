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
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if err := s.wal.LogSet(key, value); err != nil {
		return err
	}

	s.set(key, value, time.Time{})

	return nil
}

func (s *Store) set(key, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.data[key]; ok {
		s.lruList.Remove(existing.element)
	}

	element := s.lruList.PushFront(key)
	s.data[key] = entry{
		value:   value,
		expiry:  expiry,
		element: element,
	}

	if s.capacity > 0 && len(s.data) > s.capacity {
		oldest := s.lruList.Back()
		s.lruList.Remove(oldest)
		delete(s.data, oldest.Value.(string))
		s.metrics.Evictions.Add(1)
	}
}

func (s *Store) SetWithTTL(key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return s.Set(key, value)
	}

	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	expiry := time.Now().Add(ttl)

	if err := s.wal.LogSetWithTTL(key, value, expiry); err != nil {
		return err
	}

	s.set(key, value, expiry)

	return nil
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

	if err := s.wal.LogDelete(key); err != nil {
		return err
	}

	s.del(key)

	return nil
}

func (s *Store) del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.data[key]; ok {
		s.lruList.Remove(existing.element)
		delete(s.data, key)
	}
}