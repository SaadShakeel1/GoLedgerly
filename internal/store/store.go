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
}

func New() *Store {
	return &Store{
		data: make(map[string]entry),
	}
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
