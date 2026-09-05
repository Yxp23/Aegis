package router

import (
	"context"
	"testing"

	"github.com/Yxp23/aegis/internal/providers"
	"github.com/Yxp23/aegis/internal/providers/mock"
)

func benchmarkChatRequest() providers.ChatRequest {
	return providers.ChatRequest{
		Model: "mock/mock-model",
		Messages: []providers.Message{
			{
				Role:    "user",
				Content: "benchmark",
			},
		},
	}
}

func BenchmarkRouterChat(b *testing.B) {
	mockProvider := &mock.Provider{}
	r := New(mockProvider)

	req := benchmarkChatRequest()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := r.Chat(
			context.Background(),
			req,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRouterChatParallel(b *testing.B) {
	mockProvider := &mock.Provider{}
	r := New(mockProvider)

	req := benchmarkChatRequest()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := r.Chat(
				context.Background(),
				req,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
