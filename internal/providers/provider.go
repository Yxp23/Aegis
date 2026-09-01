package providers

import "context"

type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
type StreamingProvider interface {
	Provider

	StreamChat(
		ctx context.Context,
		req ChatRequest,
		onChunk StreamHandler,
	) error
}
