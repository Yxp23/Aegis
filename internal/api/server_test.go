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
