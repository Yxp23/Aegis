package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/Yxp23/aegis/internal/providers"
)

type Router struct {
	providers map[string]providers.Provider
}

func New(providerList ...providers.Provider) *Router {
	registry := make(map[string]providers.Provider)

	for _, provider := range providerList {
		registry[provider.Name()] = provider
	}

	return &Router{
		providers: registry,
	}
}

var _ providers.Provider = (*Router)(nil)

func (r *Router) Name() string {
	return "router"
}

func (r *Router) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	parts := strings.SplitN(req.Model, "/", 2)

	if len(parts) != 2 {
		return providers.ChatResponse{}, fmt.Errorf(
			"model must use provider/model format",
		)
	}

	providerName := parts[0]
	modelName := parts[1]

	provider, ok := r.providers[providerName]
	if !ok {
		return providers.ChatResponse{}, fmt.Errorf(
			"unknown provider: %s",
			providerName,
		)
	}

	req.Model = modelName

	return provider.Chat(ctx, req)
}
