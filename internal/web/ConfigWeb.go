package web

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
)

type RestRegexConfig struct {
	EventDelimiter  string   `json:"eventDelimiter"`
	FieldExtractors []string `json:"fieldExtractors"`
}

type RestParserConfig struct {
	Type        string          `json:"type"`
	RegexConfig RestRegexConfig `json:"regexConfig"`
}

type RestFileTypeConfig struct {
	Name         string           `json:"name"`
	TimeLayout   string           `json:"timeLayout"`
	ReadInterval string           `json:"readInterval"`
	Parser       RestParserConfig `json:"parser"`
}

func ToDomainFileTypeConfig(rftc RestFileTypeConfig) (string, *config.FileTypeConfig, error) {
	readInterval, err := time.ParseDuration(rftc.ReadInterval)
	if err != nil {
		return "", nil, fmt.Errorf("got error when parsing duration=%v: %w", rftc.ReadInterval, err)
	}
	eventDelimiter, err := regexp.Compile(rftc.Parser.RegexConfig.EventDelimiter)
	if err != nil {
		return "", nil, fmt.Errorf("got error when compiling regexp=%v: %w", rftc.Parser.RegexConfig.EventDelimiter, err)
	}
	fes := make([]*regexp.Regexp, len(rftc.Parser.RegexConfig.FieldExtractors))
	for i, fe := range rftc.Parser.RegexConfig.FieldExtractors {
		fes[i], err = regexp.Compile(fe)
		if err != nil {
			return "", nil, fmt.Errorf("got error when compiling regexp=%v: %w", fe, err)
		}
	}
	return rftc.Name, &config.FileTypeConfig{
		TimeLayout:   rftc.TimeLayout,
		ReadInterval: readInterval,
		ParserType:   config.ParserTypeRegex,
		Regex: &parser.RegexParserConfig{
			EventDelimiter:  eventDelimiter,
			FieldExtractors: fes,
		},
	}, nil
}

func ToRestFileTypeConfig(name string, ftc config.FileTypeConfig) RestFileTypeConfig {
	fes := make([]string, len(ftc.Regex.FieldExtractors))
	for i, fe := range ftc.Regex.FieldExtractors {
		fes[i] = fe.String()
	}
	return RestFileTypeConfig{
		Name:         name,
		TimeLayout:   ftc.TimeLayout,
		ReadInterval: ftc.ReadInterval.String(),
		Parser: RestParserConfig{
			Type: "Regex",
			RegexConfig: RestRegexConfig{
				EventDelimiter:  ftc.Regex.EventDelimiter.String(),
				FieldExtractors: fes,
			},
		},
	}
}

func addConfigEndpoints(g *gin.RouterGroup, wi *webImpl) {
	g = g.Group("config")
	g.GET("fileTypes", func(c *gin.Context) {
		fileTypes := wi.dynamicConfig.Cd("fileTypes")
		names, ok := fileTypes.Ls(false)
		if !ok {
			c.AbortWithError(500, fmt.Errorf("got error when listing keys in fileTypes"))
			return
		}
		ret := make([]RestFileTypeConfig, 0, len(names))
		for _, name := range names {
			dynamicFtc := fileTypes.Cd(name)
			ftc, err := config.FileTypeConfigFromDynamicConfig(name, dynamicFtc)
			if err != nil {
				log.Printf("got error when reading file type config for file type with name=%v: %v\n", name, err)
				continue
			}
			rftc := ToRestFileTypeConfig(name, *ftc)
			ret = append(ret, rftc)
		}
		c.JSON(200, ret)
	})

	g.POST("fileTypes", func(c *gin.Context) {
		var rftc RestFileTypeConfig
		err := json.NewDecoder(c.Request.Body).Decode(&rftc)
		if err != nil {
			c.AbortWithError(400, fmt.Errorf("got error when decoding file type config: %w", err))
			return
		}
		name, ftc, err := ToDomainFileTypeConfig(rftc)
		if err != nil {
			c.AbortWithError(400, fmt.Errorf("got error when converting file type config: %w", err))
			return
		}
		fes := make([]string, len(ftc.Regex.FieldExtractors))
		for i, fe := range ftc.Regex.FieldExtractors {
			fes[i] = fe.String()
		}
		jfe, err := json.Marshal(fes)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("failed to marshal fieldExtractors before storing them in config"))
			return
		}
		m := map[string]string{
			"fileTypes." + name + ".timeLayout":                         ftc.TimeLayout,
			"fileTypes." + name + ".readInterval":                       ftc.ReadInterval.String(),
			"fileTypes." + name + ".parser.regexConfig.eventDelimiter":  ftc.Regex.EventDelimiter.String(),
			"fileTypes." + name + ".parser.regexConfig.fieldExtractors": string(jfe),
		}
		err = wi.configRepo.SetAll(m)
		if err != nil {
			c.AbortWithError(500, fmt.Errorf("got error when setting config values: %w", err))
			return
		}
		c.JSON(200, nil)
	})
}
