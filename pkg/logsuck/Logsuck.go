package logsuck

import (
	"log/slog"

	"go.uber.org/dig"
)

type ProviderFunc func(c *dig.Container, logger *slog.Logger) error

type Plugin struct {
	Name    string
	Provide ProviderFunc

	JsonSchema func() (map[string]any, error)
}
