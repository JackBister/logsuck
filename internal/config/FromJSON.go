package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"time"
)

type jsonFileConfig struct {
	FileName       string `json:fileName`
	EventDelimiter string `json:eventDelimiter`
	ReadInterval   string `json:readInterval`
	TimeLayout     string `json:timeLayout`
}

type jsonSqliteConfig struct {
	FileName string `json:fileName`
}

type jsonWebConfig struct {
	Enabled bool   `json:enabled`
	Address string `json:address`
}

type jsonConfig struct {
	Files           []jsonFileConfig  `json:files`
	FieldExtractors []string          `json:fieldExtractors`
	Sqlite          *jsonSqliteConfig `json:sqlite`
	Web             *jsonWebConfig    `json:web`
}

var defaultConfig = Config{
	IndexedFiles: []IndexedFileConfig{},

	FieldExtractors: []*regexp.Regexp{
		regexp.MustCompile("(\\w+)=(\\w+)"),
		regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)"),
	},

	SQLite: &SqliteConfig{
		DatabaseFile: "logsuck.db",
	},

	Web: &WebConfig{
		Enabled: true,
		Address: ":8080",
	},
}

var defaultEventDelimiter = regexp.MustCompile("\n")
var defaultReadInterval = 1 * time.Second
var defaultTimeLayout = "2006/01/02 15:04:05"

func FromJSON(r io.Reader) (*Config, error) {
	var cfg jsonConfig
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config JSON: %w", err)
	}

	indexedFiles := make([]IndexedFileConfig, len(cfg.Files))
	for i, file := range cfg.Files {
		if file.FileName == "" {
			return nil, fmt.Errorf("error reading config at files[%v]: fileName is empty", i)
		}
		indexedFiles[i].Filename = file.FileName

		if file.EventDelimiter == "" {
			log.Printf("Using default event delimiter for file=%v, defaultEventDelimiter=%v\n", file.FileName, defaultEventDelimiter)
			indexedFiles[i].EventDelimiter = defaultEventDelimiter
		} else {
			ed, err := regexp.Compile(file.EventDelimiter)
			if err != nil {
				return nil, fmt.Errorf("error reading config at files[%v]: error compiling eventDelimiter regexp: %w", i, err)
			}
			indexedFiles[i].EventDelimiter = ed
		}

		if file.ReadInterval == "" {
			log.Printf("Using default read interval for file=%v, defaultReadInterval=%v\n", file.FileName, defaultReadInterval)
			indexedFiles[i].ReadInterval = defaultReadInterval
		} else {
			ri, err := time.ParseDuration(file.ReadInterval)
			if err != nil {
				return nil, fmt.Errorf("error reading config at files[%v]: error parsing readInterval duration: %w", i, err)
			}
			indexedFiles[i].ReadInterval = ri
		}

		if file.TimeLayout == "" {
			log.Printf("Using default time layout for file=%v, defaultTimeLayout=%v\n", file.FileName, defaultTimeLayout)
			indexedFiles[i].TimeLayout = defaultTimeLayout
		} else {
			indexedFiles[i].TimeLayout = file.TimeLayout
		}
	}

	var fieldExtractors []*regexp.Regexp
	if len(cfg.FieldExtractors) == 0 {
		log.Printf("Using default field extractors. defaultFieldExtractors=%v\n", defaultConfig.FieldExtractors)
		fieldExtractors = defaultConfig.FieldExtractors
	} else {
		fieldExtractors = make([]*regexp.Regexp, len(cfg.FieldExtractors))
		for i, fe := range cfg.FieldExtractors {
			re, err := regexp.Compile(fe)
			if err != nil {
				return nil, fmt.Errorf("error reading config at fieldExtractors[%v]: error compiling regexp: %w", i, err)
			}
			fieldExtractors[i] = re
		}
	}

	var sqlite *SqliteConfig
	if cfg.Sqlite == nil {
		log.Println("Using default sqlite configuration.")
		sqlite = defaultConfig.SQLite
	} else {
		sqlite = &SqliteConfig{}
		if cfg.Sqlite.FileName == "" {
			log.Printf("Using default sqlite filename. defaultFileName=%v\n", defaultConfig.SQLite.DatabaseFile)
			sqlite.DatabaseFile = defaultConfig.SQLite.DatabaseFile
		} else {
			sqlite.DatabaseFile = cfg.Sqlite.FileName
		}
	}

	var web *WebConfig
	if cfg.Web == nil {
		log.Println("Using default web configuration.")
		web = defaultConfig.Web
	} else {
		web = &WebConfig{
			Enabled: cfg.Web.Enabled,
		}
		if cfg.Web.Address == "" {
			log.Printf("Using default web address. defaultWebAddress=%v\n", defaultConfig.Web.Address)
			web.Address = defaultConfig.Web.Address
		} else {
			web.Address = cfg.Web.Address
		}
	}

	if cfg.Web != nil {
		log.Println("web", cfg.Web)
	}
	return &Config{
		IndexedFiles:    indexedFiles,
		FieldExtractors: fieldExtractors,
		SQLite:          sqlite,
		Web:             web,
	}, nil
}
