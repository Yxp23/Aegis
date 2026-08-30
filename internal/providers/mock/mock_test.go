package mock

import (
	"context"
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
)

func TestProviderChat(t *testing.T) {
	provider := &Provider{}
	req := providers.ChatRequest{
		Model: "mock-model",
		Messages: []providers.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}
	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Content != "mock: hello" {
		t.Fatalf("expected mock: hello, got %q", resp.Content)
	}
}
