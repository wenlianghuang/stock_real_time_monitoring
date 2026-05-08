package agg

import (
	"sync"
	"time"

	"stock_record/backend/internal/domain"
)

type seriesStore struct {
	mu     sync.RWMutex
	window time.Duration
	bySym  map[string][]domain.SparkPoint
}

func newSeriesStore(window time.Duration) *seriesStore {
	return &seriesStore{
		window: window,
		bySym:  make(map[string][]domain.SparkPoint),
	}
}

func (s *seriesStore) add(symbol string, t int64, p float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	points := append(s.bySym[symbol], domain.SparkPoint{T: t, P: p})
	cutoff := time.Now().Add(-s.window).UnixMilli()

	// keep last window
	i := 0
	for i < len(points) && points[i].T < cutoff {
		i++
	}
	if i > 0 {
		points = points[i:]
	}
	s.bySym[symbol] = points
}

func (s *seriesStore) getMany(symbols []string) map[string][]domain.SparkPoint {
	out := make(map[string][]domain.SparkPoint, len(symbols))
	s.mu.RLock()
	for _, sym := range symbols {
		if pts, ok := s.bySym[sym]; ok {
			cp := make([]domain.SparkPoint, len(pts))
			copy(cp, pts)
			out[sym] = cp
		}
	}
	s.mu.RUnlock()
	return out
}

