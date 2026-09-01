package router

import (
	"sync"
	"time"
)

type ProviderHealth struct {
	ConsecutiveFailures int
	Healthy             bool
	OpenUntil           time.Time
	ProbeInFlight       bool
}

type HealthTracker struct {
	mu        sync.RWMutex
	providers map[string]ProviderHealth
	threshold int
	cooldown  time.Duration
}

func NewHealthTracker(threshold int, cooldown time.Duration) *HealthTracker {
	return &HealthTracker{
		providers: make(map[string]ProviderHealth),
		threshold: threshold,
		cooldown:  cooldown,
	}
}
func (h *HealthTracker) Record(provider string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.providers[provider]
	if !exists {
		health.Healthy = true
	}
	health.ProbeInFlight = false

	if err == nil {
		health.ConsecutiveFailures = 0
		health.Healthy = true
		health.OpenUntil = time.Time{}
	} else {
		health.ConsecutiveFailures++

		if health.ConsecutiveFailures >= h.threshold {
			health.Healthy = false
			health.OpenUntil = time.Now().Add(h.cooldown)
		}
	}
	h.providers[provider] = health
}
func (h *HealthTracker) IsHealthy(provider string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health, exists := h.providers[provider]
	if !exists {
		return true
	}

	return health.Healthy
}
func (h *HealthTracker) ShouldAllow(provider string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.providers[provider]
	if !exists {
		return true
	}

	if health.Healthy {
		return true
	}

	if time.Now().Before(health.OpenUntil) {
		return false
	}

	if health.ProbeInFlight {
		return false
	}

	health.ProbeInFlight = true
	h.providers[provider] = health

	return true
}
