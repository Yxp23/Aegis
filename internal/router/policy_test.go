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
