package postgres_common

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/dig"
)

const pluginName = "@logsuck/postgres_common"

//go:embed postgres_common.schema.json
var schemaString string

type Config struct {
	ConnectionString string
}

var Plugin = logsuck.Plugin{
	Name: pluginName,
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func(cfg *Config) (*pgxpool.Pool, error) {
			pool, err := pgxpool.New(context.Background(), cfg.ConnectionString)
			if err != nil {
				return nil, err
			}
			return pool, nil
		})
		if err != nil {
			return err
		}
		err = c.Provide(func(cfg *config.Config) (*Config, error) {
			ret := Config{}

			cfgMap, ok := cfg.Plugins[pluginName].(map[string]any)
			if ok {
				if cs, ok := cfgMap["connectionString"].(string); ok {
					ret.ConnectionString = cs
				}
			}
			if ret.ConnectionString == "" {
				return nil, fmt.Errorf("got empty connectionString in postgres_common configuration")
			}
			return &ret, nil
		})
		if err != nil {
			return err
		}
		return nil
	},
	JsonSchema: func() (map[string]any, error) {
		ret := map[string]any{}
		err := json.Unmarshal([]byte(schemaString), &ret)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal postgres_common JSON schema: %w", err)
		}
		return ret, nil
	},
}
