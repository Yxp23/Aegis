package api

import (
	"encoding/json"
	"net/http"

	"github.com/Yxp23/aegis/internal/providers"
)

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []providers.Message `json:"messages"`
	Stream   bool                `json:"stream,omitempty"`
}
type chatResponse struct {
	Content string `json:"content"`
}

func chatHandler(provider providers.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(
				w,
				`{"error":"invalid request body"}`,
				http.StatusBadRequest,
			)
			return
		}

		providerReq := providers.ChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
		}

		if req.Stream {
			streamingProvider, ok := provider.(providers.StreamingProvider)
			if !ok {
				http.Error(
					w,
					`{"error":"provider does not support streaming"}`,
					http.StatusBadGateway,
				)
				return
			}

			streamChat(w, r, streamingProvider, providerReq)
			return
		}

		resp, err := provider.Chat(r.Context(), providerReq)
		if err != nil {
			http.Error(
				w,
				`{"error":"provider request failed"}`,
				http.StatusBadGateway,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(chatResponse{
			Content: resp.Content,
		})
	}
}
func streamChat(
	w http.ResponseWriter,
	r *http.Request,
	provider providers.StreamingProvider,
	req providers.ChatRequest,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	err := provider.StreamChat(
		r.Context(),
		req,
		func(chunk providers.StreamChunk) error {
			if _, err := w.Write([]byte("data: ")); err != nil {
				return err
			}

			if err := json.NewEncoder(w).Encode(chatResponse{
				Content: chunk.Content,
			}); err != nil {
				return err
			}

			if _, err := w.Write([]byte("\n")); err != nil {
				return err
			}

			flusher.Flush()
			return nil
		},
	)

	if err != nil {
		return
	}

	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
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
