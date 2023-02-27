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
	"fmt"
	"log"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/dependencyinjection"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/recipient"
	"github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"
	"go.uber.org/dig"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

var versionString string // This must be set using -ldflags "-X main.versionString=<version>" when building for --version to work

func main() {
	cmdFlags := config.ParseCommandLine()

	if cmdFlags.PrintVersion {
		if versionString == "" {
			fmt.Println("(unknown version)")
			return
		}
		fmt.Println(versionString)
		return
	}

	var err error
	var logger *zap.Logger
	if cmdFlags.LogType == "development" {
		logger, err = zap.NewDevelopment()

	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed to create Zap logger: %v\n", err)
		return
	}

	staticConfig := cmdFlags.ToConfig(logger)
	c, err := dependencyinjection.InjectionContextFromConfig(staticConfig, cmdFlags.ForceStaticConfig, logger)
	if err != nil {
		logger.Fatal("failed to create dependency injection context", zap.Error(err))
		return
	}

	err = c.Invoke(func(p struct {
		dig.In

		Ctx               context.Context
		ConfigSource      config.ConfigSource
		Publisher         events.EventPublisher
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
				logger.Fatal("got error when reading dynamic file config", zap.Error(err))
				return
			}
			reloadFileWatchers(logger.Named("reloadFileWatchers"), &watchers, indexedFiles, p.StaticConfig, &r.Cfg, p.Publisher, p.Ctx)
		}

		if p.StaticConfig.Recipient.Enabled {
			go func() {
				logger.Fatal("got error from recipient Serve method", zap.Error(p.RecipientEndpoint.Serve()))
			}()
		}

		if p.StaticConfig.Web.Enabled {
			go func() {
				logger.Fatal("got error from web Serve method", zap.Error(p.Web.Serve()))
			}()
		}

		go func() {
			for {
				<-p.ConfigSource.Changes()
				newCfg, err := p.ConfigSource.Get()
				if err != nil {
					logger.Warn("got error when reading updated dynamic file config. file and task config will not be updated", zap.Error(err))
					continue
				}
				if !p.StaticConfig.Recipient.Enabled {
					newIndexedFiles, err := indexedfiles.ReadFileConfig(&newCfg.Cfg, logger)
					if err != nil {
						logger.Warn("got error when reading updated dynamic file config. file config will not be updated", zap.Error(err))
					} else {
						reloadFileWatchers(logger.Named("reloadFileWatchers"), &watchers, newIndexedFiles, p.StaticConfig, &newCfg.Cfg, p.Publisher, p.Ctx)
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

func reloadFileWatchers(logger *zap.Logger, watchers *map[string]*files.GlobWatcher, indexedFiles []indexedfiles.IndexedFileConfig, staticConfig *config.Config, cfg *config.Config, publisher events.EventPublisher, ctx context.Context) {
	logger.Info("reloading file watchers", zap.Int("newIndexedFilesLen", len(indexedFiles)), zap.Int("oldIndexedFilesLen", len(*watchers)))
	indexedFilesMap := map[string]indexedfiles.IndexedFileConfig{}
	for _, cfg := range indexedFiles {
		indexedFilesMap[cfg.Filename] = cfg
	}
	watchersToDelete := []string{}
	// Update existing watchers and find watchers to delete
	for k, v := range *watchers {
		newCfg, ok := indexedFilesMap[k]
		if !ok {
			logger.Info("filename not found in new indexed files config. will cancel and delete watcher", zap.String("fileName", k))
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
		logger.Info("creating new watcher", zap.String("fileName", k))
		w, err := files.NewGlobWatcher(v, v.Filename, staticConfig.HostName, publisher, ctx, logger.Named("GlobWatcher"))
		if err != nil {
			logger.Warn("got error when creating GlobWatcher", zap.String("fileName", v.Filename), zap.Error(err))
			continue
		}
		(*watchers)[k] = w
	}
}
