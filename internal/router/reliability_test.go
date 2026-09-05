package router

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Yxp23/aegis/internal/providers"
)

type reliabilityPolicy struct{}

func (reliabilityPolicy) Select(
	req providers.ChatRequest,
) (Route, error) {
	return Route{
		Provider: "primary",
		Model:    "primary-model",
		Fallbacks: []Route{
			{
				Provider: "fallback",
				Model:    "fallback-model",
			},
		},
	}, nil
}

type timeoutProvider struct {
	calls atomic.Int64
}

func (p *timeoutProvider) Name() string {
	return "primary"
}

func (p *timeoutProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	p.calls.Add(1)

	<-ctx.Done()

	return providers.ChatResponse{}, ctx.Err()
}

type reliabilityFallbackProvider struct {
	calls atomic.Int64
}

func (p *reliabilityFallbackProvider) Name() string {
	return "fallback"
}

func (p *reliabilityFallbackProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	p.calls.Add(1)

	return providers.ChatResponse{
		Content: "fallback succeeded",
	}, nil
}

func TestTimeoutRetryFailoverAndCircuitBreaker(t *testing.T) {
	primary := &timeoutProvider{}
	fallback := &reliabilityFallbackProvider{}

	r := NewWithPolicy(
		reliabilityPolicy{},
		primary,
		fallback,
	)

	r.providerTimeout = 20 * time.Millisecond
	r.maxRetries = 1
	r.retryBackoff = time.Millisecond

	r.health = NewHealthTracker(
		2,
		time.Minute,
	)

	req := providers.ChatRequest{
		Model: "anything",
		Messages: []providers.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	first, err := r.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	if first.Content != "fallback succeeded" {
		t.Fatalf(
			"expected fallback response, got %q",
			first.Content,
		)
	}

	if got := primary.calls.Load(); got != 2 {
		t.Fatalf(
			"expected primary to be called twice, got %d",
			got,
		)
	}

	if got := fallback.calls.Load(); got != 1 {
		t.Fatalf(
			"expected fallback to be called once, got %d",
			got,
		)
	}

	if r.IsProviderHealthy("primary") {
		t.Fatal("expected primary circuit to be open")
	}

	second, err := r.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if second.Content != "fallback succeeded" {
		t.Fatalf(
			"expected fallback response, got %q",
			second.Content,
		)
	}

	if got := primary.calls.Load(); got != 2 {
		t.Fatalf(
			"expected circuit breaker to skip primary; got %d calls",
			got,
		)
	}

	if got := fallback.calls.Load(); got != 2 {
		t.Fatalf(
			"expected fallback to handle second request; got %d calls",
			got,
		)
	}
}
