package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	handler := NewHandler()
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
