// Copyright 2020 The Logsuck Authors
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
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/parser"
)

type Web interface {
	Serve() error
}

type webImpl struct {
	cfg       *config.Config
	eventRepo events.Repository
	jobRepo   jobs.Repository
	jobEngine *jobs.Engine
}

type webError struct {
	err  string
	code int
}

func (w webError) Error() string {
	return w.err
}

func NewWeb(cfg *config.Config, eventRepo events.Repository, jobRepo jobs.Repository, jobEngine *jobs.Engine) Web {
	return webImpl{
		cfg:       cfg,
		eventRepo: eventRepo,
		jobRepo:   jobRepo,
		jobEngine: jobEngine,
	}
}

//go:embed static/dist
var Assets embed.FS

func (wi webImpl) Serve() error {
	r := gin.Default()

	var filesys http.FileSystem
	if wi.cfg.Web.UsePackagedFiles {
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
		})
		c.Status(200)
	})

	r.GET("/search", func(c *gin.Context) {
		tpl.Execute(c.Writer, gin.H{
			"scriptSrc": "search.js",
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
		eventIds, err := wi.jobRepo.GetResults(jobId, skip, take)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		results, err := wi.eventRepo.GetByIds(eventIds, events.SortModeTimestampDesc)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		retResults := make([]events.EventWithExtractedFields, 0, len(results))
		for _, r := range results {
			fields := parser.ExtractFields(r.Raw, wi.cfg.FieldExtractors)
			retResults = append(retResults, events.EventWithExtractedFields{
				Id:        r.Id,
				Raw:       r.Raw,
				Host:      r.Host,
				Source:    r.Source,
				Timestamp: r.Timestamp,
				Fields:    fields,
			})
		}
		c.JSON(200, retResults)
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

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		c.FileFromFS(path, filesys)
	})

	log.Printf("Starting Web GUI on address='%v'\n", wi.cfg.Web.Address)
	return r.Run(wi.cfg.Web.Address)
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
