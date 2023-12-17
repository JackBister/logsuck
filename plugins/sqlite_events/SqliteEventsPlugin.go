package sqlite_events

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/sqlite_events",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewSqliteEventRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
