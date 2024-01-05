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
