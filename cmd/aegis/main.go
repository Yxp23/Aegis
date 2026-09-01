package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Yxp23/aegis/internal/api"
	"github.com/Yxp23/aegis/internal/config"
	"github.com/Yxp23/aegis/internal/providers/anthropic"
	"github.com/Yxp23/aegis/internal/providers/openai"
	"github.com/Yxp23/aegis/internal/router"
	"github.com/Yxp23/aegis/internal/telemetry"
)

func main() {
	cfg := config.Load()
	logger := telemetry.NewLogger()
	anthropicProvider := anthropic.New(cfg.AnthropicAPIKey)
	openaiProvider := openai.New(cfg.OpenAIAPIKey)
	routingPolicy := router.NewAliasPolicy(
		map[string]router.Route{
			"fast": {
				Provider: "openai",
				Model:    "gpt-5.4-mini",
				Fallbacks: []router.Route{
					{
						Provider: "anthropic",
						Model:    "claude-sonnet-4-6",
					},
				},
			},
			"quality": {
				Provider: "anthropic",
				Model:    "claude-sonnet-4-6",
				Fallbacks: []router.Route{
					{
						Provider: "openai",
						Model:    "gpt-5.4-mini",
					},
				},
			},
		},
		router.PrefixPolicy{},
	)

	provider := router.NewWithPolicy(
		routingPolicy,
		anthropicProvider,
		openaiProvider,
	)
	handler := api.NewHandler(provider)
	logger.Info("server started", "port", cfg.Port)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	logger.Info("server shutting down")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
