package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
