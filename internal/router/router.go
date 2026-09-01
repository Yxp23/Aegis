package router

import (
	"context"
	"fmt"

	"time"

	"github.com/Yxp23/aegis/internal/providers"
)

type Router struct {
	providers       map[string]providers.Provider
	policy          Policy
	stats           *Stats
	health          *HealthTracker
	providerTimeout time.Duration
	maxRetries      int
	retryBackoff    time.Duration
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
		providers:       registry,
		policy:          policy,
		stats:           NewStats(),
		health:          NewHealthTracker(3, 30*time.Second),
		providerTimeout: 15 * time.Second,
		maxRetries:      1,
		retryBackoff:    100 * time.Millisecond,
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

		resp, err := r.callProviderWithRetry(ctx, candidate, req)
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
	providerCtx, cancel := context.WithTimeout(
		ctx,
		r.providerTimeout,
	)
	defer cancel()

	start := time.Now()

	resp, err := provider.Chat(providerCtx, req)

	r.stats.Record(
		route.Provider,
		time.Since(start),
		err,
	)

	r.health.Record(route.Provider, err)

	return resp, err
}

func (r *Router) callProviderWithRetry(
	ctx context.Context,
	route Route,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		resp, err := r.callProvider(ctx, route, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			return providers.ChatResponse{}, ctx.Err()
		}

		if attempt < r.maxRetries {
			timer := time.NewTimer(r.retryBackoff)

			select {
			case <-ctx.Done():
				timer.Stop()
				return providers.ChatResponse{}, ctx.Err()

			case <-timer.C:
			}
		}
	}

	return providers.ChatResponse{}, lastErr
}

var _ providers.StreamingProvider = (*Router)(nil)

func (r *Router) streamProvider(
	ctx context.Context,
	route Route,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
	provider, ok := r.providers[route.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", route.Provider)
	}

	streamingProvider, ok := provider.(providers.StreamingProvider)
	if !ok {
		return fmt.Errorf("provider %s does not support streaming", route.Provider)
	}

	req.Model = route.Model

	start := time.Now()

	err := streamingProvider.StreamChat(
		ctx,
		req,
		onChunk,
	)

	r.stats.Record(
		route.Provider,
		time.Since(start),
		err,
	)

	r.health.Record(route.Provider, err)

	return err
}
func (r *Router) StreamChat(
	ctx context.Context,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
	route, err := r.policy.Select(req)
	if err != nil {
		return err
	}

	return r.streamProvider(
		ctx,
		route,
		req,
		onChunk,
	)
}
