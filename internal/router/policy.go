package router

import (
	"fmt"
	"strings"

	"github.com/Yxp23/aegis/internal/providers"
)

type Route struct {
	Provider  string
	Model     string
	Fallbacks []Route
}

type Policy interface {
	Select(req providers.ChatRequest) (Route, error)
}

type PrefixPolicy struct{}

func (p PrefixPolicy) Select(req providers.ChatRequest) (Route, error) {
	parts := strings.SplitN(req.Model, "/", 2)

	if len(parts) != 2 {
		return Route{}, fmt.Errorf("model must use provider/model format")
	}

	return Route{
		Provider: parts[0],
		Model:    parts[1],
	}, nil
}

type StaticPolicy struct {
	routes map[string]Route
}

func NewStaticPolicy(routes map[string]Route) *StaticPolicy {
	return &StaticPolicy{
		routes: routes,
	}
}

func (p *StaticPolicy) Select(req providers.ChatRequest) (Route, error) {
	route, ok := p.routes[req.Model]
	if !ok {
		return Route{}, fmt.Errorf("no route configured for model: %s", req.Model)
	}

	return route, nil
}

type AliasPolicy struct {
	aliases  map[string]Route
	fallback Policy
}

func NewAliasPolicy(aliases map[string]Route, fallback Policy) *AliasPolicy {
	return &AliasPolicy{
		aliases:  aliases,
		fallback: fallback,
	}
}

func (p *AliasPolicy) Select(req providers.ChatRequest) (Route, error) {
	if route, ok := p.aliases[req.Model]; ok {
		return route, nil
	}

	return p.fallback.Select(req)
}
