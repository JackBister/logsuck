package postgres_common

import (
	"context"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/postgres_common",
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
		return nil
	},
}
