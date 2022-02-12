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

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"
)

type jsonFileConfig struct {
	Filename       string `json:fileName`
	EventDelimiter string `json:eventDelimiter`
	ReadInterval   string `json:readInterval`
	TimeLayout     string `json:timeLayout`
}

type jsonForwarderConfig struct {
	Enabled           *bool  `json:enabled`
	MaxBufferedEvents *int   `json:maxBufferedEvents`
	RecipientAddress  string `json:recipientAddress`
}

type jsonRecipientConfig struct {
	Enabled     *bool             `json:enabled`
	Address     string            `json:address`
	TimeLayouts map[string]string `json:timeLayouts`
}

type jsonSqliteConfig struct {
	FileName  string `json:fileName`
	TrueBatch *bool  `json:trueBatch`
}

type jsonTaskConfig struct {
	Name     string            `json:name`
	Enabled  bool              `json:enabled`
	Interval string            `json:interval`
	Config   map[string]string `json:config`
}

type jsonTasksConfig struct {
	Tasks []jsonTaskConfig `json:tasks`
}

type jsonWebConfig struct {
	Enabled          *bool  `json:enabled`
	Address          string `json:address`
	UsePackagedFiles *bool  `json:usePackagedFiles`
	DebugMode        *bool  `json:debugMode`
}

type jsonConfig struct {
	Files           []jsonFileConfig `json:files`
	FieldExtractors []string         `json:fieldExtractors`

	HostName string `json:hostName`

	Forwarder *jsonForwarderConfig `json:forwarder`
	Recipient *jsonRecipientConfig `json:recipient`
	Sqlite    *jsonSqliteConfig    `json:sqlite`
	Tasks     *jsonTasksConfig     `json:tasks`
	Web       *jsonWebConfig       `json:web`
}

var defaultConfig = Config{
	IndexedFiles: []IndexedFileConfig{},

	FieldExtractors: []*regexp.Regexp{
		regexp.MustCompile("(\\w+)=(\\w+)"),
		regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)"),
	},

	Forwarder: &ForwarderConfig{
		Enabled:           false,
		MaxBufferedEvents: 1000000,
		RecipientAddress:  "http://localhost:8081",
	},

	Recipient: &RecipientConfig{
		Enabled: false,
		Address: ":8081",
		TimeLayouts: map[string]string{
			"DEFAULT": defaultTimeLayout,
		},
	},

	SQLite: &SqliteConfig{
		DatabaseFile: "logsuck.db",
		TrueBatch:    true,
	},

	Tasks: &TasksConfig{
		Tasks: map[string]TaskConfig{},
	},

	Web: &WebConfig{
		Enabled:          true,
		Address:          ":8080",
		UsePackagedFiles: true,
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
		if file.Filename == "" {
			return nil, fmt.Errorf("error reading config at files[%v]: pattern is empty", i)
		}
		indexedFiles[i].Filename = file.Filename

		if file.EventDelimiter == "" {
			log.Printf("Using default event delimiter for file=%v, defaultEventDelimiter=%v\n", file.Filename, defaultEventDelimiter)
			indexedFiles[i].EventDelimiter = defaultEventDelimiter
		} else {
			ed, err := regexp.Compile(file.EventDelimiter)
			if err != nil {
				return nil, fmt.Errorf("error reading config at files[%v]: error compiling eventDelimiter regexp: %w", i, err)
			}
			indexedFiles[i].EventDelimiter = ed
		}

		if file.ReadInterval == "" {
			log.Printf("Using default read interval for file=%v, defaultReadInterval=%v\n", file.Filename, defaultReadInterval)
			indexedFiles[i].ReadInterval = defaultReadInterval
		} else {
			ri, err := time.ParseDuration(file.ReadInterval)
			if err != nil {
				return nil, fmt.Errorf("error reading config at files[%v]: error parsing readInterval duration: %w", i, err)
			}
			indexedFiles[i].ReadInterval = ri
		}

		if file.TimeLayout == "" {
			log.Printf("Using default time layout for file=%v, defaultTimeLayout=%v\n", file.Filename, defaultTimeLayout)
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

	var hostName string
	if cfg.HostName != "" {
		log.Printf("Using hostName=%v\n", cfg.HostName)
		hostName = cfg.HostName
	} else {
		log.Println("No hostName in configuration, will try to get host name from operating system.")
		hostName, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting host name: %w", err)
		}
		log.Printf("Got host name from operating system. hostName=%v\n", hostName)
	}

	var forwarder *ForwarderConfig
	if cfg.Forwarder == nil {
		log.Println("Using default forwarder configuration.")
		forwarder = defaultConfig.Forwarder
	} else {
		forwarder = &ForwarderConfig{}
		if cfg.Forwarder.Enabled == nil {
			log.Println("forwarder.enabled not specified, defaulting to false")
			forwarder.Enabled = false
		} else {
			forwarder.Enabled = *cfg.Forwarder.Enabled
		}
		if cfg.Forwarder.MaxBufferedEvents == nil {
			log.Printf("Using default maxBufferedEvents for forwarder. defaultBufferedEvents=%v\n", defaultConfig.Forwarder.MaxBufferedEvents)
			forwarder.MaxBufferedEvents = defaultConfig.Forwarder.MaxBufferedEvents
		} else {
			forwarder.MaxBufferedEvents = *cfg.Forwarder.MaxBufferedEvents
		}
		if cfg.Forwarder.RecipientAddress == "" {
			log.Printf("Using default recipientAddress for forwarder. dedfaultRecipientAddress=%v\n", defaultConfig.Forwarder.RecipientAddress)
			forwarder.RecipientAddress = defaultConfig.Forwarder.RecipientAddress
		} else {
			forwarder.RecipientAddress = cfg.Forwarder.RecipientAddress
		}
	}

	var recipient *RecipientConfig
	if cfg.Recipient == nil {
		log.Println("Using default recipient configuration.")
		recipient = defaultConfig.Recipient
	} else {
		recipient = &RecipientConfig{}
		if cfg.Recipient.Enabled == nil {
			log.Println("recipient.enabled not specified, defaulting to false")
			recipient.Enabled = false
		} else {
			recipient.Enabled = *cfg.Recipient.Enabled
		}
		if cfg.Recipient.Address == "" {
			log.Printf("Using default address for recipient. defaultAddress=%v\n", defaultConfig.Recipient.Address)
			recipient.Address = defaultConfig.Recipient.Address
		} else {
			recipient.Address = cfg.Recipient.Address
		}
		if cfg.Recipient.TimeLayouts == nil {
			log.Printf("Using default time layouts for recipient. defaultTimeLayouts=%v\n", defaultConfig.Recipient.TimeLayouts)
			recipient.TimeLayouts = defaultConfig.Recipient.TimeLayouts
		} else {
			recipient.TimeLayouts = cfg.Recipient.TimeLayouts
			if _, ok := recipient.TimeLayouts["DEFAULT"]; !ok {
				log.Printf("No DEFAULT key found in recipient.timeLayouts, will add DEFAULT timeLayout '%v'\n", defaultConfig.Recipient.TimeLayouts["DEFAULT"])
				recipient.TimeLayouts["DEFAULT"] = defaultConfig.Recipient.TimeLayouts["DEFAULT"]
			}
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
		if cfg.Sqlite.TrueBatch == nil {
			log.Println("Using default TrueBatch mode. defaultTrueBatch=true")
			sqlite.TrueBatch = true
		} else {
			sqlite.TrueBatch = *cfg.Sqlite.TrueBatch
		}
	}

	var tasksConfig *TasksConfig
	if cfg.Tasks == nil {
		log.Println("Using default task configuration.")
		tasksConfig = defaultConfig.Tasks
	} else {
		tasks := make(map[string]TaskConfig, len(cfg.Tasks.Tasks))
		for _, jsonTask := range cfg.Tasks.Tasks {
			interval, err := time.ParseDuration(jsonTask.Interval)
			if err != nil {
				log.Printf("failed to parse interval for task='%s'\n", jsonTask.Name)
				continue
			}
			tasks[jsonTask.Name] = TaskConfig{
				Name:     jsonTask.Name,
				Enabled:  jsonTask.Enabled,
				Interval: interval,
				Config: NewDynamicConfig([]ConfigSource{
					NewMapConfigSource(jsonTask.Config),
				}),
			}
		}
		tasksConfig = &TasksConfig{
			Tasks: tasks,
		}
	}

	var web *WebConfig
	if cfg.Web == nil {
		log.Println("Using default web configuration.")
		web = defaultConfig.Web
		if forwarder.Enabled {
			log.Println("Disabling web GUI since forwarder is enabled.")
			web.Enabled = false
		}
	} else {
		web = &WebConfig{}
		if cfg.Web.Enabled == nil {
			if forwarder.Enabled {
				log.Println("web.enabled not specified but forwarder.enabled is true. Setting web.enabled to false")
				web.Enabled = false
			} else {
				log.Println("web.enabled not specified, defaulting to true")
				web.Enabled = true
			}
		} else {
			web.Enabled = *cfg.Web.Enabled
		}
		if cfg.Web.Address == "" {
			log.Printf("Using default web address. defaultWebAddress=%v\n", defaultConfig.Web.Address)
			web.Address = defaultConfig.Web.Address
		} else {
			web.Address = cfg.Web.Address
		}
		if cfg.Web.UsePackagedFiles == nil {
			log.Println("web.usePackagedFiles not specified, defaulting to true")
			web.UsePackagedFiles = true
		} else {
			web.UsePackagedFiles = *cfg.Web.UsePackagedFiles
		}
		if cfg.Web.DebugMode == nil {
			log.Println("web.debugMode not specified, defaulting to false")
			web.DebugMode = false
		} else {
			web.DebugMode = *cfg.Web.DebugMode
		}
	}

	return &Config{
		IndexedFiles:    indexedFiles,
		FieldExtractors: fieldExtractors,

		HostName: hostName,

		Forwarder: forwarder,
		Recipient: recipient,

		SQLite: sqlite,

		Tasks: tasksConfig,

		Web: web,
	}, nil
}
