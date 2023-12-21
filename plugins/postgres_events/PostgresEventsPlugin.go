package postgres_events

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/postgres_events",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewPostgresEventRepository)
		return err
	},
}
