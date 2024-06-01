// Copyright 2024 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
