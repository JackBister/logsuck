package web

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackbister/logsuck/internal/config"
)

func addConfigEndpoints(g *gin.RouterGroup, wi *webImpl) {
	g = g.Group("config")

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
		ctx.JSON(200, cfgJson)
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
		var jsonCfg config.JsonConfig
		err = json.NewDecoder(ctx.Request.Body).Decode(&jsonCfg)
		if err != nil {
			ctx.AbortWithError(500, fmt.Errorf("failed to decode json config: %w", err))
			return
		}
		cfgResp, err := config.FromJSON(jsonCfg)
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
