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
	"log/slog"

	internalConfig "github.com/jackbister/logsuck/internal/config"
	internalEvents "github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/forwarder"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/pipeline"
	"github.com/jackbister/logsuck/internal/recipient"
	internalTasks "github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/tasks"

	"go.uber.org/dig"
)

func InjectionContextFromConfig(cfg *config.Config, forceStaticConfig bool, logger *slog.Logger) (*dig.Container, error) {
	c := dig.New()
	err := provideBasics(c, forceStaticConfig, cfg, logger)
	if err != nil {
		return nil, err
	}

	pluginSchemas := map[string]any{}
	for _, p := range usedPlugins {
		logger.Info("Loading plugin", slog.String("pluginName", p.Name))
		err = p.Provide(c, logger)
		if err != nil {
			return nil, err
		}
		if p.JsonSchema != nil {
			schema, err := p.JsonSchema()
			if err != nil {
				return nil, err
			}
			pluginSchemas[p.Name] = schema
		}
	}
	err = c.Provide(func(p struct {
		dig.In
		Tasks []tasks.Task `group:"tasks"`
	}) (map[string]any, error) {
		taskSchemas := map[string]any{}
		for _, t := range p.Tasks {
			taskSchemas[t.Name()] = t.ConfigSchema()
		}
		return internalConfig.CreateSchema(pluginSchemas, taskSchemas)
	}, dig.Name("configSchema"))
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
	err = providePipelines(c, logger)
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
	if cfg.Recipient.Enabled {
		err = c.Provide(recipient.NewRecipientEndpoint)
		if err != nil {
			return nil, err
		}
	} else {
		err = c.Provide(files.NewGlobWatcherCoordinator)
		if err != nil {
			return nil, err
		}
	}
	err = c.Provide(web.NewWeb)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func provideBasics(c *dig.Container, forceStaticConfig bool, cfg *config.Config, logger *slog.Logger) error {
	err := c.Provide(func() *slog.Logger {
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

func providePublisher(c *dig.Container) error {
	return c.Invoke(func(cfg *config.Config) error {
		if cfg.Forwarder.Enabled {
			err := c.Provide(forwarder.ForwardingEventPublisher)
			if err != nil {
				return err
			}
		} else {
			err := c.Provide(internalEvents.BatchedRepositoryPublisher)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func provideConfigSource(c *dig.Container, logger *slog.Logger) error {
	return c.Invoke(func(p struct {
		dig.In

		Cfg               *config.Config
		ForceStaticConfig bool `name:"forceStaticConfig"`
	}) error {
		if p.ForceStaticConfig {
			logger.Info("Static configuration is forced. Configuration will not be saved to database and will only be read from the JSON configuration file. Remove the forceStaticConfig flag from the command line or configuration file in order to use dynamic configuration.")
			err := c.Provide(func() config.Source {
				return &config.StaticSource{
					Config: *p.Cfg,
				}
			})
			if err != nil {
				return err
			}
		} else if p.Cfg.Forwarder.Enabled {
			err := c.Provide(forwarder.NewRemoteConfigSource)
			if err != nil {
				return err
			}
			return nil
		} else {
			err := c.Provide(func(p struct {
				dig.In
				ConfigRepo config.Repository
			}) config.Source {
				return p.ConfigRepo
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func providePipelines(c *dig.Container, logger *slog.Logger) error {
	err := c.Provide(pipeline.NewPipelineCompiler)
	if err != nil {
		return err
	}
	return nil
}

func provideTasks(c *dig.Container, logger *slog.Logger) error {
	err := c.Provide(internalTasks.NewTaskManager)
	if err != nil {
		return err
	}
	return nil
}

func provideEnumProviders(c *dig.Container, logger *slog.Logger) error {
	err := c.Provide(internalTasks.NewTaskEnumProvider, dig.Group("enumProviders"))
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
