package sqlite

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/sqlite",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewSqliteConfigRepository)
		if err != nil {
			return err
		}
		err = c.Provide(NewSqliteJobRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
