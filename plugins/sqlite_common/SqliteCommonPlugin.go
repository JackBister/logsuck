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

package sqlite_common

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/config"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/dig"
)

const pluginName = "@logsuck/sqlite_common"

//go:embed sqlite_common.schema.json
var schemaString string

type Config struct {
	FileName string
}

var Plugin = logsuck.Plugin{
	Name: pluginName,
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func() string {
			return "sqlite3"
		}, dig.Name("sqlDriver"))
		if err != nil {
			return err
		}
		err = c.Provide(func(cfg *config.Config) *Config {
			ret := Config{
				FileName: "logsuck.db",
			}
			cfgMap, ok := cfg.Plugins[pluginName].(map[string]any)
			if !ok {
				return &ret
			}
			if fns, ok := cfgMap["fileName"].(string); ok {
				ret.FileName = fns
			}
			return &ret
		})
		if err != nil {
			return err
		}
		err = c.Provide(func(cfg *Config) string {
			additionalSqliteParameters := "?_journal_mode=WAL"
			if cfg.FileName == ":memory:" {
				// cache=shared breaks DeleteOldEventsTask. But not having it breaks everything in :memory: mode.
				// So we set cache=shared for :memory: mode and assume people will not need to delete old tasks in that mode.
				additionalSqliteParameters += "&cache=shared"
			}
			return "file:" + cfg.FileName + additionalSqliteParameters
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
	JsonSchema: func() (map[string]any, error) {
		ret := map[string]any{}
		err := json.Unmarshal([]byte(schemaString), &ret)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal sqlite_common JSON schema: %w", err)
		}
		return ret, nil
	},
}
