package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"bufio"
	"strings"

	"github.com/Yxp23/aegis/internal/providers"
)

const endpoint = "https://api.openai.com/v1/responses"

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
	return "openai"
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model  string    `json:"model"`
	Input  []message `json:"input"`
	Stream bool      `json:"stream,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type outputItem struct {
	Type    string         `json:"type"`
	Content []contentBlock `json:"content"`
}

type response struct {
	Output []outputItem `json:"output"`
}
type streamEvent struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
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
		Model: req.Model,
		Input: messages,
	})
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("marshal openai request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("create openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return providers.ChatResponse{}, fmt.Errorf("openai request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return providers.ChatResponse{}, fmt.Errorf(
			"openai returned status %d",
			httpResp.StatusCode,
		)
	}
	var result response

	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return providers.ChatResponse{}, fmt.Errorf("decode openai response: %w", err)
	}

	for _, item := range result.Output {
		if item.Type != "message" {
			continue
		}

		for _, block := range item.Content {
			if block.Type == "output_text" {
				return providers.ChatResponse{
					Content: block.Text,
				}, nil
			}
		}
	}

	return providers.ChatResponse{}, fmt.Errorf("openai returned no text content")

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
		Model:  req.Model,
		Input:  messages,
		Stream: true,
	})
	if err != nil {
		return fmt.Errorf("marshal openai streaming request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create openai streaming request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai streaming request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf(
			"openai returned status %d",
			httpResp.StatusCode,
		)
	}

	scanner := bufio.NewScanner(httpResp.Body)

	// Streaming events can occasionally be larger than Scanner's
	// default token size.
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(
			strings.TrimPrefix(line, "data:"),
		)

		if data == "" || data == "[DONE]" {
			continue
		}

		var event streamEvent

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf(
				"decode openai stream event: %w",
				err,
			)
		}

		if event.Type != "response.output_text.delta" {
			continue
		}

		if event.Delta == "" {
			continue
		}

		if err := onChunk(providers.StreamChunk{
			Content: event.Delta,
		}); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read openai stream: %w", err)
	}

	return nil
}

var _ providers.StreamingProvider = (*Provider)(nil)
