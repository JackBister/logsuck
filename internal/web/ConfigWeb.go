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

package web

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/jackbister/logsuck/pkg/logsuck/config"
)

func addConfigEndpoints(g *gin.RouterGroup, wi *webImpl) {
	g = g.Group("config")

	g.GET("/schema", func(ctx *gin.Context) {
		ctx.JSON(200, wi.configSchema)
	})
	g.GET("/enums/:name", func(ctx *gin.Context) {
		providerName, _ := ctx.Params.Get("name")
		provider, ok := wi.enumProviders[providerName]
		if !ok {
			ctx.AbortWithError(500, fmt.Errorf("failed to get enum provider with name=%v", providerName))
			return
		}
		values, err := provider.Values()
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to get values from enum provider with name=%v: %w", providerName, err))
		}
		ctx.JSON(200, values)
	})

	g.GET("", func(ctx *gin.Context) {
		cfg, err := wi.configSource.Get()
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to read config: %w", err))
			return
		}
		cfgJson, err := config.ToJSON(&cfg.Cfg)
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to convert config to json: %w", err))
			return
		}
		ctx.Data(200, "application/json", cfgJson)
	})

	g.POST("", func(ctx *gin.Context) {
		dynamicCfgResp, err := wi.configSource.Get()
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("faild to read configuration: %w", err))
			return
		}
		if dynamicCfgResp.Cfg.ForceStaticConfig {
			ctx.AbortWithError(400, fmt.Errorf("cannot save configuration because forceStaticConfig is enabled"))
			return
		}
		jsonCfg, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to read json config: %w", err))
			return
		}
		cfgResp, err := config.FromJSON(jsonCfg, wi.logger)
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to convert config from json: %w", err))
			return
		}
		err = wi.configRepo.Upsert(cfgResp)
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("got error when upserting config: %w", err))
			return
		}
		ctx.String(200, "ok")
	})

}
