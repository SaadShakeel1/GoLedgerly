package store

import (
	"sync"
	"time"
)

type entry struct {
	value  string
	expiry time.Time
}

type Store struct {
	mu   sync.RWMutex
	data map[string]entry
	stop chan struct{}
}

func New() *Store {
	s := &Store{
		data: make(map[string]entry),
		stop: make(chan struct{}),
	}
	go s.startExpiryLoop(250 * time.Millisecond)
	return s
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
			delete(s.data, key)
		}
	}

}

func (s *Store) Stop() {
	close(s.stop)
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry{value: value}
}

func (s *Store) SetWithTTL(key, value string, ttl time.Duration) {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry{value: value, expiry: exp}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	if !ok {
		s.mu.RUnlock()
		return "", false
	}

	if !e.expiry.IsZero() && time.Now().After(e.expiry) {
		s.mu.RUnlock()

		s.mu.Lock()
		defer s.mu.Unlock()

		e2, ok2 := s.data[key]
		if !ok2 {
			return "", false
		}

		if !e2.expiry.IsZero() && time.Now().After(e2.expiry) {
			delete(s.data, key)
			return "", false
		}

		return e2.value, true
	}

	val := e.value
	s.mu.RUnlock()
	return val, true
}
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}
