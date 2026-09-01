package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
func TestProviderStreamChat(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("unexpected authorization header: %q", got)
			}

			w.Header().Set("Content-Type", "text/event-stream")

			w.Write([]byte(
				"data: {\"type\":\"response.created\"}\n\n",
			))

			w.Write([]byte(
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello \"}\n\n",
			))

			w.Write([]byte(
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"from openai\"}\n\n",
			))

			w.Write([]byte(
				"data: {\"type\":\"response.completed\"}\n\n",
			))
		}),
	)
	defer server.Close()

	provider := New("test-key")
	provider.baseURL = server.URL

	var chunks []string

	err := provider.StreamChat(
		context.Background(),
		providers.ChatRequest{
			Model: "gpt-test",
			Messages: []providers.Message{
				{
					Role:    "user",
					Content: "hello",
				},
			},
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
	want := "hello from openai"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
