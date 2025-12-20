package mem

import (
	"sync"
)

type KeyRecord struct {
	TgUserID     int64
	CountryCode  string
	OutlineKeyID string
	AccessURL    string
}

type Store struct {
	mu   sync.RWMutex
	byID map[int64]KeyRecord
}

func New() *Store {
	return &Store{byID: map[int64]KeyRecord{}}
}

func (s *Store) Get(tgUserID int64) (KeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.byID[tgUserID]
	return v, ok
}

func (s *Store) Put(v KeyRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[v.TgUserID] = v
}
