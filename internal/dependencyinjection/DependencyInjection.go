// Copyright 2023 Jack Bister
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

package dependencyinjection

import (
	"context"
	"database/sql"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/forwarder"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/recipient"
	"github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

func InjectionContextFromConfig(cfg *config.Config, forceStaticConfig bool, logger *zap.Logger) (*dig.Container, error) {
	c := dig.New()
	err := provideBasics(c, forceStaticConfig, cfg, logger)
	if err != nil {
		return nil, err
	}
	err = provideSqlite(c)
	if err != nil {
		return nil, err
	}
	err = providePublisher(c)
	if err != nil {
		return nil, err
	}
	err = provideConfigSource(c, logger)
	if err != nil {
		return nil, err
	}
	err = provideTasks(c, logger)
	if err != nil {
		return nil, err
	}
	err = provideEnumProviders(c, logger)
	if err != nil {
		return nil, err
	}
	err = c.Provide(jobs.NewEngine)
	if err != nil {
		return nil, err
	}
	err = c.Provide(recipient.NewRecipientEndpoint)
	if err != nil {
		return nil, err
	}
	err = c.Provide(web.NewWeb)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func provideBasics(c *dig.Container, forceStaticConfig bool, cfg *config.Config, logger *zap.Logger) error {
	err := c.Provide(func() *zap.Logger {
		return logger
	})
	if err != nil {
		return err
	}
	err = c.Provide(func() *config.Config {
		return cfg
	})
	if err != nil {
		return err
	}
	err = c.Provide(context.Background)
	if err != nil {
		return err
	}
	err = c.Provide(func(cfg *config.Config) bool {
		return forceStaticConfig || cfg.ForceStaticConfig
	}, dig.Name("forceStaticConfig"))
	if err != nil {
		return err
	}
	return nil
}

func provideSqlite(c *dig.Container) error {
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
	err = c.Provide(config.NewSqliteConfigRepository)
	if err != nil {
		return err
	}
	err = c.Provide(events.SqliteRepository)
	if err != nil {
		return err
	}
	err = c.Provide(jobs.SqliteRepository)
	if err != nil {
		return err
	}
	return nil
}

func providePublisher(c *dig.Container) error {
	return c.Invoke(func(cfg *config.Config) error {
		if cfg.Forwarder.Enabled {
			err := c.Provide(forwarder.ForwardingEventPublisher)
			if err != nil {
				return err
			}
		} else {
			err := c.Provide(events.BatchedRepositoryPublisher)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func provideConfigSource(c *dig.Container, logger *zap.Logger) error {
	return c.Invoke(func(p struct {
		dig.In

		Cfg               *config.Config
		ForceStaticConfig bool `name:"forceStaticConfig"`
	}) error {
		if p.ForceStaticConfig {
			logger.Info("Static configuration is forced. Configuration will not be saved to database and will only be read from the JSON configuration file. Remove the forceStaticConfig flag from the command line or configuration file in order to use dynamic configuration.")
			err := c.Provide(func() config.ConfigSource {
				return &config.StaticConfigSource{
					Config: *p.Cfg,
				}
			})
			if err != nil {
				return err
			}
		}
		if p.Cfg.Forwarder.Enabled {
			err := c.Provide(forwarder.NewRemoteConfigSource)
			if err != nil {
				return err
			}
			return nil
		}
		err := c.Provide(func(p struct {
			dig.In
			ConfigRepo config.ConfigRepository
		}) config.ConfigSource {
			return p.ConfigRepo
		})
		if err != nil {
			return err
		}
		return nil
	})
}

func provideTasks(c *dig.Container, logger *zap.Logger) error {
	err := c.Provide(tasks.NewDeleteOldEventsTask, dig.Group("tasks"))
	if err != nil {
		return err
	}
	err = c.Provide(tasks.NewTaskManager)
	if err != nil {
		return err
	}
	return nil
}

func provideEnumProviders(c *dig.Container, logger *zap.Logger) error {
	err := c.Provide(tasks.NewTaskEnumProvider, dig.Group("enumProviders"))
	if err != nil {
		return err
	}
	err = c.Provide(web.NewFileTypeEnumProvider, dig.Group("enumProviders"))
	if err != nil {
		return err
	}
	err = c.Provide(web.NewFileEnumProvider, dig.Group("enumProviders"))
	if err != nil {
		return err
	}
	err = c.Provide(web.NewHostTypeEnumProvider, dig.Group("enumProviders"))
	if err != nil {
		return err
	}
	return nil
}
