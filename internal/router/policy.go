package router

import (
	"fmt"
	"strings"

	"github.com/Yxp23/aegis/internal/providers"
)

type Route struct {
	Provider string
	Model    string
}

type Policy interface {
	Select(req providers.ChatRequest) (Route, error)
}

type PrefixPolicy struct{}

func (p PrefixPolicy) Select(req providers.ChatRequest) (Route, error) {
	parts := strings.SplitN(req.Model, "/", 2)

	if len(parts) != 2 {
		return Route{}, fmt.Errorf("model must use provider/model format")
	}

	return Route{
		Provider: parts[0],
		Model:    parts[1],
	}, nil
}
