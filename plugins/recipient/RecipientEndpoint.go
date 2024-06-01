// Copyright 2022 Jack Bister
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

package recipient

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/indexedfiles"
	"github.com/jackbister/logsuck/pkg/logsuck/parser"
	"github.com/jackbister/logsuck/pkg/logsuck/rpc"
	"github.com/jackbister/logsuck/pkg/logsuck/util"

	"go.uber.org/dig"
)

type RecipientEndpoint struct {
	configSource config.Source
	staticConfig *config.Config
	repo         events.Repository

	logger *slog.Logger
}

type RecipientEndpointParams struct {
	dig.In

	ConfigSource config.Source
	StaticConfig *config.Config
	Repo         events.Repository
	Logger       *slog.Logger
}

func NewRecipientEndpoint(p RecipientEndpointParams) events.Reader {
	return &RecipientEndpoint{configSource: p.ConfigSource, staticConfig: p.StaticConfig, repo: p.Repo, logger: p.Logger}
}

func (er *RecipientEndpoint) Start() error {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(util.NewGinSlogger(slog.LevelInfo, *er.logger))
	r.SetTrustedProxies(nil)

	r.GET("/v1/config", func(c *gin.Context) {
		cfg, err := er.configSource.Get()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		cfgJson, err := config.ToJSON(&cfg.Cfg)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, rpc.ConfigResponse{
			Modified: &cfg.Modified,
			Config:   cfgJson,
		})
	})

	r.POST("/v1/receiveEvents", func(c *gin.Context) {
		var req rpc.ReceiveEventsRequest
		err := json.NewDecoder(c.Request.Body).Decode(&req)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("failed to decode JSON: %w", err))
			return
		}
		cfg, err := er.configSource.Get()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		indexedFileConfigs, err := indexedfiles.ReadFileConfig(&cfg.Cfg, er.logger)
		if err != nil {
			// TODO:
		}

		sourceToConfig := map[string]*indexedfiles.IndexedFileConfig{}
		for _, evt := range req.Events {
			if _, ok := sourceToConfig[evt.Source]; ok {
				continue
			}
			for i, ifc := range indexedFileConfigs {
				absGlob, err := filepath.Abs(ifc.Filename)
				if err != nil {
					// TODO:
					continue
				}
				absSource, err := filepath.Abs(evt.Source)
				if err != nil {
					// TODO:
					continue
				}
				if m, err := filepath.Match(absGlob, absSource); err == nil && m {
					sourceToConfig[evt.Source] = &indexedFileConfigs[i]
					goto nextfile
				}
			}
		nextfile:
		}
		processed := make([]events.Event, len(req.Events))
		for i, evt := range req.Events {
			processed[i] = events.Event{
				Raw:      evt.Raw,
				Host:     evt.Host,
				Source:   evt.Source,
				SourceId: evt.SourceId,
				Offset:   evt.Offset,
			}

			ifc, ok := sourceToConfig[evt.Source]
			if !ok {
				// TODO:
			}

			fields, err := parser.ExtractFields(strings.ToLower(evt.Raw), ifc.FileParser)
			if err != nil {
				er.logger.Warn("failed to extract fields when getting timestamp, will use current time as timestamp",
					slog.Any("error", err))
				processed[i].Timestamp = time.Now()
			} else if t, ok := fields["_time"]; ok {
				parsed, err := parser.ParseTime(ifc.TimeLayout, t)
				if err != nil {
					er.logger.Warn("failed to parse _time field, will use current time as timestamp",
						slog.Any("error", err))
					processed[i].Timestamp = time.Now()
				} else {
					processed[i].Timestamp = parsed
				}
			} else {
				er.logger.Warn("no _time field extracted for event, got fields, will use current time as timestamp",
					slog.String("eventRaw", evt.Raw),
					slog.Any("fields", fields))
				processed[i].Timestamp = time.Now()
			}
		}
		err = er.repo.AddBatch(processed)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("failed to add events to repository: %w", err))
			return
		}
		c.Status(200)
	})

	er.logger.Info("Starting recipient",
		slog.String("address", er.staticConfig.Recipient.Address))
	return r.Run(er.staticConfig.Recipient.Address)
}
