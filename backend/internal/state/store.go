package state

import (
	"sort"
	"sync"

	"stock_record/backend/internal/domain"
)

type Store struct {
	mu     sync.RWMutex
	quotes map[string]domain.QuoteState
}

func NewStore() *Store {
	return &Store{
		quotes: make(map[string]domain.QuoteState),
	}
}

func (s *Store) UpsertQuote(q domain.QuoteState) {
	s.mu.Lock()
	s.quotes[q.Symbol] = q
	s.mu.Unlock()
}

func (s *Store) GetQuote(symbol string) (domain.QuoteState, bool) {
	s.mu.RLock()
	q, ok := s.quotes[symbol]
	s.mu.RUnlock()
	return q, ok
}

func (s *Store) Snapshot(symbols []string) map[string]domain.QuoteState {
	out := make(map[string]domain.QuoteState, len(symbols))
	s.mu.RLock()
	for _, sym := range symbols {
		if q, ok := s.quotes[sym]; ok {
			out[sym] = q
		}
	}
	s.mu.RUnlock()
	return out
}

func (s *Store) Symbols() []string {
	s.mu.RLock()
	out := make([]string, 0, len(s.quotes))
	for k := range s.quotes {
		out = append(out, k)
	}
	s.mu.RUnlock()
	sort.Strings(out)
	return out
}

