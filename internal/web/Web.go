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

package web

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/parser"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

type Web interface {
	Serve() error
}

type webImpl struct {
	configSource  config.ConfigSource
	configRepo    config.ConfigRepository
	staticConfig  *config.Config
	eventRepo     events.Repository
	jobRepo       jobs.Repository
	jobEngine     *jobs.Engine
	enumProviders map[string]EnumProvider

	logger *zap.Logger
}

type webError struct {
	err  string
	code int
}

func (w webError) Error() string {
	return w.err
}

type WebParams struct {
	dig.In

	ConfigSource config.ConfigSource
	ConfigRepo   config.ConfigRepository
	StaticConfig *config.Config
	EventRepo    events.Repository
	JobRepo      jobs.Repository
	JobEngine    *jobs.Engine
	Logger       *zap.Logger

	EnumProviders []EnumProvider `group:"enumProviders"`
}

func NewWeb(p WebParams) Web {
	enumProviders := make(map[string]EnumProvider, len(p.EnumProviders))
	for _, e := range p.EnumProviders {
		enumProviders[e.Name()] = e
	}
	return webImpl{
		staticConfig: p.StaticConfig,
		configSource: p.ConfigSource,
		configRepo:   p.ConfigRepo,
		eventRepo:    p.EventRepo,
		jobRepo:      p.JobRepo,
		jobEngine:    p.JobEngine,

		enumProviders: enumProviders,

		logger: p.Logger,
	}
}

//go:embed static/dist
var Assets embed.FS

func (wi webImpl) Serve() error {
	if wi.staticConfig.Web.DebugMode {
		wi.logger.Info("web.debugMode enabled. Will enable Gin debug mode. To disable, remove the debugMode key in the web object in your JSON config or set it to false.")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.SetTrustedProxies(nil)

	var filesys http.FileSystem
	if wi.staticConfig.Web.UsePackagedFiles {
		assets, err := fs.Sub(Assets, "static/dist")
		if err != nil {
			return fmt.Errorf("failed to Sub into static/dist directory: %w", err)
		}
		filesys = http.FS(assets)
	} else {
		filesys = http.Dir("internal/web/static/dist")
	}

	tpl, err := parseTemplate(filesys)
	if err != nil {
		return err
	}

	r.GET("/", func(c *gin.Context) {
		tpl.Execute(c.Writer, gin.H{
			"scriptSrc": "home.js",
			"styleSrc":  "home.css",
		})
		c.Status(200)
	})

	r.GET("/config", func(c *gin.Context) {
		tpl.Execute(c.Writer, gin.H{
			"scriptSrc": "config.js",
			"styleSrc":  "config.css",
		})
		c.Status(200)
	})

	r.GET("/search", func(c *gin.Context) {
		tpl.Execute(c.Writer, gin.H{
			"scriptSrc": "search.js",
			"styleSrc":  "search.css",
		})
		c.Status(200)
	})

	g := r.Group("api/v1")
	g.POST("/startJob", func(c *gin.Context) {
		searchString := c.Query("searchString")
		startTime, endTime, wErr := parseTimeParametersGin(c)
		if wErr != nil {
			c.AbortWithError(wErr.code, wErr)
			return
		}
		id, err := wi.jobEngine.StartJob(strings.TrimSpace(searchString), startTime, endTime)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, id)
	})

	g.POST("/abortJob", func(c *gin.Context) {
		jobId, err := strconv.ParseInt(c.Query("jobId"), 10, 64)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		err = wi.jobEngine.Abort(jobId)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.Status(200)
	})

	g.GET("/jobStats", func(c *gin.Context) {
		jobId, err := strconv.ParseInt(c.Query("jobId"), 10, 64)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		job, err := wi.jobRepo.Get(jobId)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		fieldCount, err := wi.jobRepo.GetFieldOccurences(jobId)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		numMatched, err := wi.jobRepo.GetNumMatchedEvents(jobId)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, gin.H{
			"State":            job.State,
			"FieldCount":       fieldCount,
			"NumMatchedEvents": numMatched,
		})
	})

	g.GET("/jobResults", func(c *gin.Context) {
		jobId, err := strconv.ParseInt(c.Query("jobId"), 10, 64)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		skip, err := strconv.Atoi(c.Query("skip"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		take, err := strconv.Atoi(c.Query("take"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		job, err := wi.jobRepo.Get(jobId)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		eventResults, err := wi.getJobResults(job, skip, take)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		tableRows, err := wi.jobRepo.GetTableResults(job.Id, skip, take)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, gin.H{
			"resultType":  job.OutputType,
			"columnOrder": job.ColumnOrder,
			"events":      eventResults,
			"tableRows":   tableRows,
		})
	})

	g.GET("/jobFieldStats", func(c *gin.Context) {
		jobId, err := strconv.ParseInt(c.Query("jobId"), 10, 64)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
		fieldName, ok := c.GetQuery("fieldName")
		if !ok {
			c.AbortWithStatus(400)
			return
		}
		values, err := wi.jobRepo.GetFieldValues(jobId, fieldName)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, values)
	})

	addConfigEndpoints(g, &wi)

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		c.FileFromFS(path, filesys)
	})

	wi.logger.Info("Starting Web GUI", zap.String("address", wi.staticConfig.Web.Address))
	return r.Run(wi.staticConfig.Web.Address)
}

func (wi *webImpl) getJobResults(job *jobs.Job, skip, take int) ([]events.EventWithExtractedFields, error) {
	eventIds, err := wi.jobRepo.GetResults(job.Id, skip, take)
	if err != nil {
		return nil, err
	}
	results, err := wi.eventRepo.GetByIds(eventIds, job.SortMode)
	if err != nil {
		return nil, err
	}
	cfg, err := wi.configSource.Get()
	if err != nil {
		return nil, err
	}
	indexedFileConfigs, err := indexedfiles.ReadFileConfig(&cfg.Cfg, wi.logger)
	if err != nil {
		return nil, err
	}
	sourceToIfc := getSourceToIndexedFileConfig(results, indexedFileConfigs)
	retResults := make([]events.EventWithExtractedFields, 0, len(results))
	for _, r := range results {
		ifc, ok := sourceToIfc[r.Source]
		if !ok {
			// TODO: default
		}
		fields, _ := parser.ExtractFields(r.Raw, ifc.FileParser)
		retResults = append(retResults, events.EventWithExtractedFields{
			Id:        r.Id,
			Raw:       r.Raw,
			Host:      r.Host,
			Source:    r.Source,
			SourceId:  r.SourceId,
			Timestamp: r.Timestamp,
			Fields:    fields,
		})
	}
	return retResults, nil
}

func parseTemplate(fs http.FileSystem) (*template.Template, error) {
	f, err := fs.Open("template.html")
	if err != nil {
		return nil, fmt.Errorf("failed to open template.html: %w", err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from template.html: %ww", err)
	}
	tpl, err := template.New("template.html").Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template.html: %w", err)
	}
	return tpl, nil
}

func parseTimeParametersGin(c *gin.Context) (*time.Time, *time.Time, *webError) {
	relativeTime, hasRelativeTime := c.GetQuery("relativeTime")
	absoluteStart, hasAbsoluteStart := c.GetQuery("startTime")
	absoluteEnd, hasAbsoluteEnd := c.GetQuery("endTime")

	if hasRelativeTime {
		relative, err := time.ParseDuration(relativeTime)
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing relativeTime: " + err.Error(),
				code: 400,
			}
		}
		startTime := time.Now().Add(relative)
		return &startTime, nil, nil
	}
	var startTime *time.Time
	var endTime *time.Time
	if hasAbsoluteStart {
		t, err := time.Parse(time.RFC3339, absoluteStart)
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing startTime: " + err.Error(),
				code: 400,
			}
		}
		startTime = &t
	}
	if hasAbsoluteEnd {
		t, err := time.Parse(time.RFC3339, absoluteEnd)
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing endTime: " + err.Error(),
				code: 400,
			}
		}
		endTime = &t
	}

	return startTime, endTime, nil
}

func getSourceToIndexedFileConfig(evts []events.EventWithId, indexedFileConfigs []indexedfiles.IndexedFileConfig) map[string]*indexedfiles.IndexedFileConfig {
	sourceToConfig := map[string]*indexedfiles.IndexedFileConfig{}
	for _, evt := range evts {
		if _, ok := sourceToConfig[evt.Source]; ok {
			continue
		}
		for i, ifc := range indexedFileConfigs {
			absGlob, err := filepath.Abs(ifc.Filename)
			if err != nil {
				// TODO:
				continue
			}
			if m, err := filepath.Match(absGlob, evt.Source); err == nil && m {
				sourceToConfig[evt.Source] = &indexedFileConfigs[i]
				goto nextfile
			}
		}
	nextfile:
	}
	return sourceToConfig
}
