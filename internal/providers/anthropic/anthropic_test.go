package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
)

func TestProviderChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("expected x-api-key header")
		}

		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Fatalf("expected anthropic-version header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
		"content": [
			{
				"type": "text",
				"text": "hello from claude"
			}
		]
	}`))
	}))
	defer server.Close()
	provider := New("test-key")
	provider.baseURL = server.URL
	req := providers.ChatRequest{
		Model: "claude-test",
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
	if resp.Content != "hello from claude" {
		t.Fatalf("expected hello from claude, got %q", resp.Content)
	}
}
