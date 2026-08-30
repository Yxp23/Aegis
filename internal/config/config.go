package config

import "os"

type Config struct {
	Port            string
	AnthropicAPIKey string
}

func Load() Config {
	port := os.Getenv("AEGIS_PORT")

	if port == "" {
		port = "8080"
	}

	return Config{
		Port:            port,
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
	}
}
