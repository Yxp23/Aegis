package router

import "sync"

type ProviderHealth struct {
	ConsecutiveFailures int
	Healthy             bool
}

type HealthTracker struct {
	mu        sync.RWMutex
	providers map[string]ProviderHealth
	threshold int
}

func NewHealthTracker(threshold int) *HealthTracker {
	return &HealthTracker{
		providers: make(map[string]ProviderHealth),
		threshold: threshold,
	}
}
func (h *HealthTracker) Record(provider string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.providers[provider]
	if !exists {
		health.Healthy = true
	}

	if err == nil {
		health.ConsecutiveFailures = 0
		health.Healthy = true
	} else {
		health.ConsecutiveFailures++

		if health.ConsecutiveFailures >= h.threshold {
			health.Healthy = false
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
