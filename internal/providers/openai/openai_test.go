package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
)

func TestProviderChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"output": [
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "hello from openai"
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	provider := New("test-key")
	provider.baseURL = server.URL

	req := providers.ChatRequest{
		Model: "test-model",
		Messages: []providers.Message{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Content != "hello from openai" {
		t.Fatalf("unexpected response: %q", resp.Content)
	}
}
