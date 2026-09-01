package router

import (
	"sync"
	"time"
)

type ProviderStats struct {
	Requests     int64
	Errors       int64
	TotalLatency time.Duration
}

type Stats struct {
	mu        sync.RWMutex
	providers map[string]ProviderStats
}

func NewStats() *Stats {
	return &Stats{
		providers: make(map[string]ProviderStats),
	}
}
func (s *Stats) Record(provider string, latency time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := s.providers[provider]

	stats.Requests++
	stats.TotalLatency += latency

	if err != nil {
		stats.Errors++
	}

	s.providers[provider] = stats
}

func (s *Stats) Snapshot(provider string) ProviderStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.providers[provider]
}
func (s ProviderStats) AverageLatency() time.Duration {
	if s.Requests == 0 {
		return 0
	}

	return s.TotalLatency / time.Duration(s.Requests)
}

func (s ProviderStats) ErrorRate() float64 {
	if s.Requests == 0 {
		return 0
	}

	return float64(s.Errors) / float64(s.Requests)
}
