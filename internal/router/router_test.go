package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
)

type testProvider struct {
	providerName string
}

func (p *testProvider) Name() string {
	return p.providerName
}

func (p *testProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	return providers.ChatResponse{
		Content: p.providerName + ":" + req.Model,
	}, nil
}

func TestRouterRoutesToProvider(t *testing.T) {
	openaiProvider := &testProvider{providerName: "openai"}
	anthropicProvider := &testProvider{providerName: "anthropic"}

	r := New(openaiProvider, anthropicProvider)

	resp, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "openai/gpt-test",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "openai:gpt-test" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}
}

func TestRouterRejectsInvalidModel(t *testing.T) {
	r := New(&testProvider{providerName: "openai"})

	_, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "gpt-test",
	})

	if err == nil {
		t.Fatal("expected error for model without provider prefix")
	}
}

func TestRouterRejectsUnknownProvider(t *testing.T) {
	r := New(&testProvider{providerName: "openai"})

	_, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "unknown/test-model",
	})

	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

type fixedPolicy struct {
	route Route
}

func (p fixedPolicy) Select(req providers.ChatRequest) (Route, error) {
	return p.route, nil
}

func TestRouterUsesInjectedPolicy(t *testing.T) {
	openaiProvider := &testProvider{providerName: "openai"}
	anthropicProvider := &testProvider{providerName: "anthropic"}

	r := NewWithPolicy(
		fixedPolicy{
			route: Route{
				Provider: "anthropic",
				Model:    "claude-test",
			},
		},
		openaiProvider,
		anthropicProvider,
	)

	resp, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "anything",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "anthropic:claude-test" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}
}
func TestRouterRecordsProviderStats(t *testing.T) {
	openaiProvider := &testProvider{providerName: "openai"}

	r := New(openaiProvider)

	_, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "openai/gpt-test",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	stats := r.ProviderStats("openai")

	if stats.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", stats.Requests)
	}

	if stats.Errors != 0 {
		t.Fatalf("expected 0 errors, got %d", stats.Errors)
	}

	anthropicStats := r.ProviderStats("anthropic")

	if anthropicStats.Requests != 0 {
		t.Fatalf(
			"expected 0 anthropic requests, got %d",
			anthropicStats.Requests,
		)
	}
}

type failingProvider struct {
	providerName string
}

func (p *failingProvider) Name() string {
	return p.providerName
}

func (p *failingProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	return providers.ChatResponse{}, fmt.Errorf("provider failed")
}
func TestRouterRecordsProviderErrors(t *testing.T) {
	openaiProvider := &failingProvider{providerName: "openai"}

	r := New(openaiProvider)

	_, err := r.Chat(context.Background(), providers.ChatRequest{
		Model: "openai/gpt-test",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}

	stats := r.ProviderStats("openai")

	if stats.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", stats.Requests)
	}

	if stats.Errors != 1 {
		t.Fatalf("expected 1 error, got %d", stats.Errors)
	}
}
func TestRouterTracksProviderHealth(t *testing.T) {
	openaiProvider := &failingProvider{providerName: "openai"}

	r := New(openaiProvider)

	for range 3 {
		_, _ = r.Chat(context.Background(), providers.ChatRequest{
			Model: "openai/gpt-test",
		})
	}

	if r.IsProviderHealthy("openai") {
		t.Fatal("expected openai to be unhealthy after 3 failures")
	}
}
func TestRouterFallsBackWhenPrimaryFails(t *testing.T) {
	openaiProvider := &failingProvider{
		providerName: "openai",
	}

	anthropicProvider := &testProvider{
		providerName: "anthropic",
	}

	r := NewWithPolicy(
		fixedPolicy{
			route: Route{
				Provider: "openai",
				Model:    "gpt-test",
				Fallbacks: []Route{
					{
						Provider: "anthropic",
						Model:    "claude-test",
					},
				},
			},
		},
		openaiProvider,
		anthropicProvider,
	)

	resp, err := r.Chat(
		context.Background(),
		providers.ChatRequest{
			Model: "anything",
		},
	)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "anthropic:claude-test" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}

	openaiStats := r.ProviderStats("openai")
	anthropicStats := r.ProviderStats("anthropic")

	if openaiStats.Requests != 1 {
		t.Fatalf("expected 1 openai request, got %d", openaiStats.Requests)
	}

	if anthropicStats.Requests != 1 {
		t.Fatalf("expected 1 anthropic request, got %d", anthropicStats.Requests)
	}
}
