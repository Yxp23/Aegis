package mock

import (
	"context"
	"strings"
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
func TestProviderStreamChat(t *testing.T) {
	provider := &Provider{}

	var chunks []string

	err := provider.StreamChat(
		context.Background(),
		providers.ChatRequest{
			Model: "mock-model",
		},
		func(chunk providers.StreamChunk) error {
			chunks = append(chunks, chunk.Content)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}

	got := strings.Join(chunks, "")
	want := "mock: streaming response"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
