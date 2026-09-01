package router

import (
	"context"
	"fmt"
	"testing"

	"time"

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
	r.maxRetries = 0

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
	r.maxRetries = 0

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
func TestRouterSkipsUnhealthyPrimary(t *testing.T) {
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

	for range 3 {
		_, err := r.Chat(
			context.Background(),
			providers.ChatRequest{Model: "anything"},
		)
		if err != nil {
			t.Fatalf("unexpected failover error: %v", err)
		}
	}

	openaiBefore := r.ProviderStats("openai").Requests

	resp, err := r.Chat(
		context.Background(),
		providers.ChatRequest{Model: "anything"},
	)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "anthropic:claude-test" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}

	openaiAfter := r.ProviderStats("openai").Requests

	if openaiAfter != openaiBefore {
		t.Fatalf(
			"expected unhealthy openai provider to be skipped",
		)
	}
}

type recoveringProvider struct {
	providerName      string
	failuresRemaining int
}

func (p *recoveringProvider) Name() string {
	return p.providerName
}

func (p *recoveringProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	if p.failuresRemaining > 0 {
		p.failuresRemaining--
		return providers.ChatResponse{}, fmt.Errorf("temporary failure")
	}

	return providers.ChatResponse{
		Content: p.providerName + ":" + req.Model,
	}, nil
}
func TestRouterRetriesProviderAfterCooldown(t *testing.T) {
	openaiProvider := &recoveringProvider{
		providerName:      "openai",
		failuresRemaining: 3,
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
	r.maxRetries = 0

	// Use a short cooldown so the test doesn't wait 30 seconds.
	r.health = NewHealthTracker(3, 10*time.Millisecond)

	for range 3 {
		_, err := r.Chat(
			context.Background(),
			providers.ChatRequest{Model: "anything"},
		)
		if err != nil {
			t.Fatalf("unexpected failover error: %v", err)
		}
	}

	if r.IsProviderHealthy("openai") {
		t.Fatal("expected openai to be unhealthy")
	}

	time.Sleep(20 * time.Millisecond)

	resp, err := r.Chat(
		context.Background(),
		providers.ChatRequest{Model: "anything"},
	)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "openai:gpt-test" {
		t.Fatalf("expected recovered openai response, got %q", resp.Content)
	}

	if !r.IsProviderHealthy("openai") {
		t.Fatal("expected openai to become healthy again")
	}
}

type slowProvider struct {
	providerName string
}

func (p *slowProvider) Name() string {
	return p.providerName
}

func (p *slowProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	select {
	case <-time.After(100 * time.Millisecond):
		return providers.ChatResponse{
			Content: "too slow",
		}, nil

	case <-ctx.Done():
		return providers.ChatResponse{}, ctx.Err()
	}
}
func TestRouterTimesOutSlowProvider(t *testing.T) {
	provider := &slowProvider{
		providerName: "openai",
	}

	r := New(provider)
	r.maxRetries = 0
	r.providerTimeout = 10 * time.Millisecond

	_, err := r.Chat(
		context.Background(),
		providers.ChatRequest{
			Model: "openai/gpt-test",
		},
	)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	stats := r.ProviderStats("openai")

	if stats.Errors != 1 {
		t.Fatalf("expected 1 provider error, got %d", stats.Errors)
	}
}
func TestRouterRetriesProviderBeforeFailover(t *testing.T) {
	openaiProvider := &recoveringProvider{
		providerName:      "openai",
		failuresRemaining: 1,
	}

	r := New(openaiProvider)
	r.maxRetries = 1
	r.retryBackoff = time.Millisecond

	resp, err := r.Chat(
		context.Background(),
		providers.ChatRequest{
			Model: "openai/gpt-test",
		},
	)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "openai:gpt-test" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}

	stats := r.ProviderStats("openai")

	if stats.Requests != 2 {
		t.Fatalf("expected 2 attempts, got %d", stats.Requests)
	}

	if stats.Errors != 1 {
		t.Fatalf("expected 1 failed attempt, got %d", stats.Errors)
	}
}
