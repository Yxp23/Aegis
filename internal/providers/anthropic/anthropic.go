package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"io"

	"bufio"
	"strings"

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
	Stream    bool      `json:"stream,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type response struct {
	Content []contentBlock `json:"content"`
}
type streamDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type streamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type streamEvent struct {
	Type  string      `json:"type"`
	Delta streamDelta `json:"delta"`
	Error streamError `json:"error"`
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
func (p *Provider) StreamChat(
	ctx context.Context,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
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
		Stream:    true,
	})
	if err != nil {
		return fmt.Errorf("marshal anthropic streaming request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create anthropic streaming request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("anthropic streaming request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf(
			"anthropic returned status %d",
			httpResp.StatusCode,
		)
	}

	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(
			strings.TrimPrefix(line, "data:"),
		)

		if data == "" {
			continue
		}

		var event streamEvent

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf(
				"decode anthropic stream event: %w",
				err,
			)
		}

		if event.Type == "error" {
			return fmt.Errorf(
				"anthropic stream error: %s",
				event.Error.Message,
			)
		}

		if event.Type != "content_block_delta" {
			continue
		}

		if event.Delta.Type != "text_delta" {
			continue
		}

		if event.Delta.Text == "" {
			continue
		}

		if err := onChunk(providers.StreamChunk{
			Content: event.Delta.Text,
		}); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read anthropic stream: %w", err)
	}

	return nil
}

var _ providers.StreamingProvider = (*Provider)(nil)
