package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"
	"fmt"
	"github.com/Yxp23/aegis/internal/providers"
	"github.com/Yxp23/aegis/internal/providers/mock"
)

func TestHealthHandler(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	expectedBody := `{"status":"ok"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, rec.Body.String())
	}
}

func TestChatHandler(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)
	body := strings.NewReader(`{
	"model": "mock-model",
	"messages": [
		{
			"role": "user",
			"content": "hello"
		}
	]
}`)
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		body,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	expectedBody := `{"content":"mock: hello"}` + "\n"

	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, rec.Body.String())
	}
}
func TestChatHandlerStreaming(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)

	body := `{
		"model":"mock/mock-model",
		"messages":[
			{"role":"user","content":"hello"}
		],
		"stream":true
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	got := recorder.Body.String()

	if !strings.Contains(got, `data: {"content":"mock: "}`) {
		t.Fatalf("missing first stream chunk: %q", got)
	}

	if !strings.Contains(got, `data: {"content":"streaming "}`) {
		t.Fatalf("missing second stream chunk: %q", got)
	}

	if !strings.Contains(got, `data: {"content":"response"}`) {
		t.Fatalf("missing final stream chunk: %q", got)
	}

	if !strings.Contains(got, "data: [DONE]") {
		t.Fatalf("missing stream completion marker: %q", got)
	}
}

type partialStreamProvider struct{}

func (p *partialStreamProvider) Name() string {
	return "partial"
}

func (p *partialStreamProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	return providers.ChatResponse{}, fmt.Errorf("not implemented")
}

func (p *partialStreamProvider) StreamChat(
	ctx context.Context,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
	if err := onChunk(providers.StreamChunk{
		Content: "partial",
	}); err != nil {
		return err
	}

	return fmt.Errorf("stream failed")
}
func TestStreamingErrorAfterOutputStarted(t *testing.T) {
	provider := &partialStreamProvider{}
	handler := NewHandler(provider)

	body := `{
		"model":"partial/test",
		"messages":[
			{"role":"user","content":"hello"}
		],
		"stream":true
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	got := recorder.Body.String()

	if !strings.Contains(got, `data: {"content":"partial"}`) {
		t.Fatalf("missing streamed chunk: %q", got)
	}

	if !strings.Contains(got, "event: error") {
		t.Fatalf("missing stream error event: %q", got)
	}

	if strings.Contains(got, "data: [DONE]") {
		t.Fatalf("unexpected completion marker after stream failure: %q", got)
	}
}

type failingStreamProvider struct{}

func (p *failingStreamProvider) Name() string {
	return "failing"
}

func (p *failingStreamProvider) Chat(
	ctx context.Context,
	req providers.ChatRequest,
) (providers.ChatResponse, error) {
	return providers.ChatResponse{}, fmt.Errorf("provider failed")
}

func (p *failingStreamProvider) StreamChat(
	ctx context.Context,
	req providers.ChatRequest,
	onChunk providers.StreamHandler,
) error {
	return fmt.Errorf("stream failed before output")
}
func TestStreamingErrorBeforeOutputStarted(t *testing.T) {
	provider := &failingStreamProvider{}
	handler := NewHandler(provider)

	body := `{
		"model":"failing/test",
		"messages":[
			{"role":"user","content":"hello"}
		],
		"stream":true
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf(
			"expected status 502, got %d",
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		`{"error":"provider stream failed"}`,
	) {
		t.Fatalf(
			"unexpected response: %q",
			recorder.Body.String(),
		)
	}
}
func TestChatHandlerRejectsWrongMethod(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/chat/completions",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status 405, got %d",
			recorder.Code,
		)
	}
}
func TestChatHandlerRejectsMissingModel(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)

	body := `{
		"messages":[
			{"role":"user","content":"hello"}
		]
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}
func TestChatHandlerRejectsMissingMessages(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)

	body := `{
		"model":"mock/mock-model"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}
func TestChatHandlerErrorsAreJSON(t *testing.T) {
	provider := &mock.Provider{}
	handler := NewHandler(provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/chat/completions",
		nil,
	)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf(
			"expected application/json content type, got %q",
			got,
		)
	}
}
