package sqlite_common

import (
	"database/sql"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/config"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/sqlite_common",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func() string {
			return "sqlite3"
		}, dig.Name("sqlDriver"))
		if err != nil {
			return err
		}
		err = c.Provide(func(cfg *config.Config) string {
			additionalSqliteParameters := "?_journal_mode=WAL"
			if cfg.SQLite.DatabaseFile == ":memory:" {
				// cache=shared breaks DeleteOldEventsTask. But not having it breaks everything in :memory: mode.
				// So we set cache=shared for :memory: mode and assume people will not need to delete old tasks in that mode.
				additionalSqliteParameters += "&cache=shared"
			}
			return "file:" + cfg.SQLite.DatabaseFile + additionalSqliteParameters
		}, dig.Name("sqlDataSourceName"))
		if err != nil {
			return err
		}
		err = c.Provide(func(p struct {
			dig.In

			DriverName     string `name:"sqlDriver"`
			DataSourceName string `name:"sqlDataSourceName"`
		}) (*sql.DB, error) {
			db, err := sql.Open(p.DriverName, p.DataSourceName)
			if err != nil {
				return nil, err
			}
			return db, nil
		})
		if err != nil {
			return err
		}
		return nil
	},
}
