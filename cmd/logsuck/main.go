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
	"github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"

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

		Ctx          context.Context
		ConfigSource config.Source
		Publisher    events.Publisher
		StaticConfig *config.Config
		EventReader  events.Reader
		TaskManager  *tasks.TaskManager
		Web          web.Web
	}) {
		go func() {
			logger.Error("got error from recipient Serve method", slog.Any("error", p.EventReader.Start()))
			os.Exit(1)
		}()

		if p.StaticConfig.Web.Enabled {
			go func() {
				logger.Error("got error from web Serve method", slog.Any("error", p.Web.Serve()))
				os.Exit(1)
			}()
		}

		p.TaskManager.Start()
	})
	if err != nil {
		log.Fatal(err)
	}

	select {}
}
