package api

import (
	"encoding/json"
	"net/http"

	"github.com/Yxp23/aegis/internal/providers"
	"log/slog"
)

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []providers.Message `json:"messages"`
	Stream   bool                `json:"stream,omitempty"`
}
type chatResponse struct {
	Content string `json:"content"`
}

func chatHandler(
	provider providers.Provider,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(
				w,
				http.StatusMethodNotAllowed,
				"method not allowed",
			)
			return
		}

		var req chatRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"invalid request body",
			)
			return
		}

		if req.Model == "" {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"model is required",
			)
			return
		}

		if len(req.Messages) == 0 {
			writeJSONError(
				w,
				http.StatusBadRequest,
				"at least one message is required",
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
				writeJSONError(
					w,
					http.StatusBadGateway,
					"provider does not support streaming",
				)
				return
			}

			streamChat(
				w,
				r,
				streamingProvider,
				providerReq,
				logger,
			)
			return
		}

		resp, err := provider.Chat(
			r.Context(),
			providerReq,
		)
		if err != nil {
			logger.Error(
				"provider request failed",
				"error", err,
				"model", req.Model,
			)

			writeJSONError(
				w,
				http.StatusBadGateway,
				"provider request failed",
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(chatResponse{
			Content: resp.Content,
		}); err != nil {
			return
		}
	}
}
func streamChat(
	w http.ResponseWriter,
	r *http.Request,
	provider providers.StreamingProvider,
	req providers.ChatRequest,
	logger *slog.Logger,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"streaming unsupported",
		)
		return
	}

	started := false

	err := provider.StreamChat(
		r.Context(),
		req,
		func(chunk providers.StreamChunk) error {
			if !started {
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				started = true
			}

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
		logger.Error(
			"provider stream failed",
			"error", err,
			"model", req.Model,
		)

		if !started {
			w.Header().Set("Content-Type", "application/json")
			writeJSONError(
				w,
				http.StatusBadGateway,
				"provider stream failed",
			)
			return

		}

		w.Write([]byte(
			"event: error\ndata: {\"error\":\"stream interrupted\"}\n\n",
		))
		flusher.Flush()
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
	return NewHandlerWithLogger(
		provider,
		slog.Default(),
	)
}

func NewHandlerWithLogger(
	provider providers.Provider,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc(
		"/v1/chat/completions",
		chatHandler(provider, logger),
	)

	return mux
}
func writeJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
