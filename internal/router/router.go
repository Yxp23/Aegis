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
	}
}

var _ providers.Provider = (*Router)(nil)

func (r *Router) Name() string {
	return "router"
}

func (r *Router) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	route, err := r.policy.Select(req)
	if err != nil {
		return providers.ChatResponse{}, err
	}

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

	return resp, err
}
func (r *Router) ProviderStats(provider string) ProviderStats {
	return r.stats.Snapshot(provider)
}
