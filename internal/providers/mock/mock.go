package mock

import (
	"context"

	"github.com/Yxp23/aegis/internal/providers"
)

var _ providers.Provider = (*Provider)(nil)

type Provider struct{}

func (p *Provider) Name() string {
	return "mock"
}
func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	content := "mock response"

	if len(req.Messages) > 0 {
		content = "mock: " + req.Messages[len(req.Messages)-1].Content
	}

	return providers.ChatResponse{
		Content: content,
	}, nil
}

var _ providers.StreamingProvider = (*Provider)(nil)

func (p *Provider) StreamChat(
	ctx context.Context,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
	chunks := []string{
		"mock: ",
		"streaming ",
		"response",
	}

	for _, content := range chunks {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := onChunk(providers.StreamChunk{
			Content: content,
		}); err != nil {
			return err
		}
	}

	return nil
}
