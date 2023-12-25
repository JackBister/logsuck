package sqlite_events

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/config"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/dig"
)

const pluginName = "@logsuck/sqlite_events"

//go:embed sqlite_events.schema.json
var schemaString string

type Config struct {
	TrueBatch bool
}

var Plugin = logsuck.Plugin{
	Name: pluginName,
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewSqliteEventRepository)
		if err != nil {
			return err
		}
		err = c.Provide(func(cfg *config.Config) *Config {
			ret := Config{
				TrueBatch: true,
			}
			cfgMap, ok := cfg.Plugins[pluginName].(map[string]any)
			if !ok {
				return &ret
			}
			if fns, ok := cfgMap["trueBatch"].(bool); ok {
				ret.TrueBatch = fns
			}
			return &ret
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
			return nil, fmt.Errorf("failed to unmarshal sqlite_events JSON schema: %w", err)
		}
		return ret, nil
	},
}
