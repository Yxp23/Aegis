package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"io"

	"github.com/Yxp23/aegis/internal/providers"
)

type Provider struct {
	apiKey  string
	client  *http.Client
	baseURL string
}

func New(apiKey string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		client:  &http.Client{},
		baseURL: endpoint,
	}
}

func (p *Provider) Name() string {
	return "anthropic"
}

const endpoint = "https://api.anthropic.com/v1/messages"

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type response struct {
	Content []contentBlock `json:"content"`
}

var _ providers.Provider = (*Provider)(nil)

func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	messages := make([]message, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	body, err := json.Marshal(request{
		Model:     req.Model,
		MaxTokens: 1024,
		Messages:  messages,
	})
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("marshal anthropic request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("create anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)

		return providers.ChatResponse{}, fmt.Errorf(
			"anthropic returned status %d: %s",
			httpResp.StatusCode,
			string(body),
		)
	}
	var result response

	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return providers.ChatResponse{}, fmt.Errorf("decode anthropic response: %w", err)
	}

	if len(result.Content) == 0 {
		return providers.ChatResponse{}, fmt.Errorf("anthropic returned no content")
	}

	return providers.ChatResponse{
		Content: result.Content[0].Text,
	}, nil
}
