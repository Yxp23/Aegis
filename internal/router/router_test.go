package router

import (
	"context"
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
