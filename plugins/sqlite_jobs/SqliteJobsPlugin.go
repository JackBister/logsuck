package sqlite_jobs

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/sqlite_jobs",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewSqliteJobRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
