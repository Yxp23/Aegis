package router

import (
	"context"
	"fmt"

	"time"

	"github.com/Yxp23/aegis/internal/providers"
)

type Router struct {
	providers map[string]providers.Provider
	policy    Policy
	stats     *Stats
	health    *HealthTracker
}

func New(providerList ...providers.Provider) *Router {
	return NewWithPolicy(PrefixPolicy{}, providerList...)
}

func NewWithPolicy(policy Policy, providerList ...providers.Provider) *Router {
	registry := make(map[string]providers.Provider)

	for _, provider := range providerList {
		registry[provider.Name()] = provider
	}

	return &Router{
		providers: registry,
		policy:    policy,
		stats:     NewStats(),
		health:    NewHealthTracker(3, 30*time.Second),
	}
}

var _ providers.Provider = (*Router)(nil)

func (r *Router) Name() string {
	return "router"
}

func (r *Router) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	route, err := r.policy.Select(req)
	if err != nil {
		return providers.ChatResponse{}, err
	}

	routes := append([]Route{route}, route.Fallbacks...)

	var lastErr error

	for _, candidate := range routes {
		if !r.health.ShouldAllow(candidate.Provider) {
			continue
		}

		resp, err := r.callProvider(ctx, candidate, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	if lastErr != nil {
		return providers.ChatResponse{}, fmt.Errorf(
			"all provider routes failed: %w",
			lastErr,
		)
	}

	return providers.ChatResponse{}, fmt.Errorf(
		"no healthy provider routes available",
	)
}

func (r *Router) ProviderStats(provider string) ProviderStats {
	return r.stats.Snapshot(provider)
}
func (r *Router) IsProviderHealthy(provider string) bool {
	return r.health.IsHealthy(provider)
}
func (r *Router) callProvider(
	ctx context.Context,
	route Route,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	provider, ok := r.providers[route.Provider]
	if !ok {
		return providers.ChatResponse{}, fmt.Errorf(
			"unknown provider: %s",
			route.Provider,
		)
	}

	req.Model = route.Model

	start := time.Now()

	resp, err := provider.Chat(ctx, req)

	r.stats.Record(
		route.Provider,
		time.Since(start),
		err,
	)

	r.health.Record(route.Provider, err)

	return resp, err
}
