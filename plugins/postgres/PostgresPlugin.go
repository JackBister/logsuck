package postgres

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/postgres",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewPostgresConfigRepository)
		if err != nil {
			return err
		}
		err = c.Provide(NewPostgresJobRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
