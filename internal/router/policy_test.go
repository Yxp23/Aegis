package router

import (
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
)

func TestPrefixPolicySelect(t *testing.T) {
	policy := PrefixPolicy{}

	route, err := policy.Select(providers.ChatRequest{
		Model: "openai/gpt-test",
	})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	if route.Provider != "openai" {
		t.Fatalf("unexpected provider: %q", route.Provider)
	}

	if route.Model != "gpt-test" {
		t.Fatalf("unexpected model: %q", route.Model)
	}
}

func TestPrefixPolicyRejectsInvalidModel(t *testing.T) {
	policy := PrefixPolicy{}

	_, err := policy.Select(providers.ChatRequest{
		Model: "gpt-test",
	})

	if err == nil {
		t.Fatal("expected error for model without provider prefix")
	}
}

func TestStaticPolicySelect(t *testing.T) {
	policy := NewStaticPolicy(map[string]Route{
		"fast": {
			Provider: "openai",
			Model:    "gpt-test",
		},
	})

	route, err := policy.Select(providers.ChatRequest{
		Model: "fast",
	})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	if route.Provider != "openai" {
		t.Fatalf("unexpected provider: %q", route.Provider)
	}

	if route.Model != "gpt-test" {
		t.Fatalf("unexpected model: %q", route.Model)
	}
}

func TestAliasPolicyFallsBackToPrefixPolicy(t *testing.T) {
	policy := NewAliasPolicy(
		map[string]Route{
			"fast": {
				Provider: "openai",
				Model:    "gpt-test",
			},
		},
		PrefixPolicy{},
	)

	aliasRoute, err := policy.Select(providers.ChatRequest{
		Model: "fast",
	})
	if err != nil {
		t.Fatalf("alias Select returned error: %v", err)
	}

	if aliasRoute.Provider != "openai" || aliasRoute.Model != "gpt-test" {
		t.Fatalf("unexpected alias route: %+v", aliasRoute)
	}

	prefixRoute, err := policy.Select(providers.ChatRequest{
		Model: "anthropic/claude-test",
	})
	if err != nil {
		t.Fatalf("fallback Select returned error: %v", err)
	}

	if prefixRoute.Provider != "anthropic" || prefixRoute.Model != "claude-test" {
		t.Fatalf("unexpected fallback route: %+v", prefixRoute)
	}
}
