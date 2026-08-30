package api

import (
	"encoding/json"
	"net/http"

	"github.com/Yxp23/aegis/internal/providers"
)

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []providers.Message `json:"messages"`
}
type chatResponse struct {
	Content string `json:"content"`
}

func chatHandler(provider providers.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		resp, err := provider.Chat(r.Context(), providers.ChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
		})
		if err != nil {
			http.Error(w, `{"error":"provider request failed"}`, http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			Content: resp.Content,
		})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func NewHandler(provider providers.Provider) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/v1/chat/completions", chatHandler(provider))

	return mux
}
