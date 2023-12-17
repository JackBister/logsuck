package postgres

import (
	"context"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/postgres",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func() (*pgxpool.Pool, error) {
			pool, err := pgxpool.New(context.Background(), "postgres://postgres:password@localhost:5432/postgres")
			if err != nil {
				return nil, err
			}
			return pool, nil
		})
		if err != nil {
			return err
		}
		err = c.Provide(NewPostgresConfigRepository)
		if err != nil {
			return err
		}
		err = c.Provide(NewPostgresJobRepository)
		if err != nil {
			return err
		}
		err = c.Invoke(func(pool *pgxpool.Pool) error {
			row := pool.QueryRow(context.Background(), "SELECT 1")
			var x int
			err = row.Scan(&x)
			if err != nil {
				return err
			}
			logger.Info("Test query got x", slog.Int("x", x))
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	},
}
