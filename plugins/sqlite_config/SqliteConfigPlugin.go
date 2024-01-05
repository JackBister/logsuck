package sqlite_config

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/sqlite_config",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewSqliteConfigRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
