// Copyright 2021 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	internalConfig "github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/dependencyinjection"
	internalEvents "github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/recipient"
	"github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"

	"github.com/jackbister/logsuck/pkg/logsuck/config"

	"go.uber.org/dig"
)

var versionString string // This must be set using -ldflags "-X main.versionString=<version>" when building for --version to work

func main() {
	cmdFlags := internalConfig.ParseCommandLine()

	if cmdFlags.PrintVersion {
		if versionString == "" {
			fmt.Println("(unknown version)")
			return
		}
		fmt.Println(versionString)
		return
	}

	var err error
	var logger *slog.Logger
	if cmdFlags.PrintJsonSchema {
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	} else if cmdFlags.LogType == "development" {
		logger = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		))
	} else {
		logger = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			},
		))
	}

	staticConfig := cmdFlags.ToConfig(logger)
	c, err := dependencyinjection.InjectionContextFromConfig(staticConfig, cmdFlags.ForceStaticConfig, logger)
	if err != nil {
		logger.Error("failed to create dependency injection context", slog.Any("error", err))
		return
	}

	if cmdFlags.PrintJsonSchema {
		err = c.Invoke(func(p struct {
			dig.In

			ConfigSchema map[string]any `name:"configSchema"`
		}) {
			b, err := json.MarshalIndent(p.ConfigSchema, "", "  ")
			if err != nil {
				logger.Error("Failed to marshal config schema", slog.Any("error", err))
				panic(err)
			}
			fmt.Print(string(b))
			os.Exit(0)
		})
		if err != nil {
			panic(err)
		}
	}

	err = c.Invoke(func(p struct {
		dig.In

		Ctx               context.Context
		ConfigSource      config.ConfigSource
		Publisher         internalEvents.EventPublisher
		StaticConfig      *config.Config
		RecipientEndpoint *recipient.RecipientEndpoint
		TaskManager       *tasks.TaskManager
		Web               web.Web
	}) {
		watchers := map[string]*files.GlobWatcher{}

		if p.StaticConfig.Recipient.Enabled {
			logger.Info("recipient is enabled, will not read any files on this host.")
		} else {
			r, _ := p.ConfigSource.Get()
			indexedFiles, err := indexedfiles.ReadFileConfig(&r.Cfg, logger)
			if err != nil {
				logger.Error("got error when reading dynamic file config", slog.Any("error", err))
				os.Exit(1)
				return
			}
			reloadFileWatchers(logger, &watchers, indexedFiles, p.StaticConfig, &r.Cfg, p.Publisher, p.Ctx)
		}

		if p.StaticConfig.Recipient.Enabled {
			go func() {
				logger.Error("got error from recipient Serve method", slog.Any("error", p.RecipientEndpoint.Serve()))
				os.Exit(1)
			}()
		}

		if p.StaticConfig.Web.Enabled {
			go func() {
				logger.Error("got error from web Serve method", slog.Any("error", p.Web.Serve()))
				os.Exit(1)
			}()
		}

		go func() {
			for {
				<-p.ConfigSource.Changes()
				newCfg, err := p.ConfigSource.Get()
				if err != nil {
					logger.Warn("got error when reading updated dynamic file config. file and task config will not be updated", slog.Any("error", err))
					continue
				}
				if !p.StaticConfig.Recipient.Enabled {
					newIndexedFiles, err := indexedfiles.ReadFileConfig(&newCfg.Cfg, logger)
					if err != nil {
						logger.Warn("got error when reading updated dynamic file config. file config will not be updated", slog.Any("error", err))
					} else {
						reloadFileWatchers(logger, &watchers, newIndexedFiles, p.StaticConfig, &newCfg.Cfg, p.Publisher, p.Ctx)
					}
				}
				p.TaskManager.UpdateConfig(newCfg.Cfg)
			}
		}()
	})
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

func reloadFileWatchers(logger *slog.Logger, watchers *map[string]*files.GlobWatcher, indexedFiles []indexedfiles.IndexedFileConfig, staticConfig *config.Config, cfg *config.Config, publisher internalEvents.EventPublisher, ctx context.Context) {
	logger.Info("reloading file watchers", slog.Int("newIndexedFilesLen", len(indexedFiles)), slog.Int("oldIndexedFilesLen", len(*watchers)))
	indexedFilesMap := map[string]indexedfiles.IndexedFileConfig{}
	for _, cfg := range indexedFiles {
		indexedFilesMap[cfg.Filename] = cfg
	}
	watchersToDelete := []string{}
	// Update existing watchers and find watchers to delete
	for k, v := range *watchers {
		newCfg, ok := indexedFilesMap[k]
		if !ok {
			logger.Info("filename not found in new indexed files config. will cancel and delete watcher", slog.String("fileName", k))
			v.Cancel()
			watchersToDelete = append(watchersToDelete, k)
			continue
		}
		v.UpdateConfig(newCfg)
	}

	// delete watchers that do not exist in the new config
	for _, k := range watchersToDelete {
		delete(*watchers, k)
	}

	// Add new watchers
	for k, v := range indexedFilesMap {
		_, ok := (*watchers)[k]
		if ok {
			continue
		}
		logger.Info("creating new watcher", slog.String("fileName", k))
		w, err := files.NewGlobWatcher(v, v.Filename, staticConfig.HostName, publisher, ctx, logger)
		if err != nil {
			logger.Warn("got error when creating GlobWatcher", slog.String("fileName", v.Filename), slog.Any("error", err))
			continue
		}
		(*watchers)[k] = w
	}
}
