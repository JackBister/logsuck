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
	"fmt"
	"log"
	"os"
	"time"
)

type jsonHostConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type jsonForwarderConfig struct {
	Enabled            *bool  `json:"enabled"`
	MaxBufferedEvents  *int   `json:"maxBufferedEvents"`
	RecipientAddress   string `json:"recipientAddress"`
	ConfigPollInterval string `json:"configPollInterval"`
}

type jsonRecipientConfig struct {
	Enabled *bool  `json:"enabled"`
	Address string `json:"address"`
}

type jsonSqliteConfig struct {
	FileName  string `json:"fileName"`
	TrueBatch *bool  `json:"trueBatch"`
}

type jsonWebConfig struct {
	Enabled          *bool  `json:"enabled"`
	Address          string `json:"address"`
	UsePackagedFiles *bool  `json:"usePackagedFiles"`
	DebugMode        *bool  `json:"debugMode"`
}

type jsonRegexFileTypeParserConfig struct {
	EventDelimiter  string   `json:"eventDelimiter"`
	FieldExtractors []string `json:"fieldExtractors"`
}

type jsonFileTypeParserConfig struct {
	Type string `json:"type"`

	RegexConfig *jsonRegexFileTypeParserConfig `json:"regexConfig"`
}

type jsonFileConfig struct {
	Filename  string   `json:"fileName"`
	FileTypes []string `json:"fileTypes"`
}

type jsonFileTypeConfig struct {
	Name         string                    `json:"name"`
	TimeLayout   string                    `json:"timeLayout"`
	ReadInterval string                    `json:"readInterval"`
	Parser       *jsonFileTypeParserConfig `json:"parser"`
}

type jsonHostTypeFileConfig struct {
	FileName string `json:"fileName"`
}

type jsonHostTypeConfig struct {
	Name  string                   `json:"name"`
	Files []jsonHostTypeFileConfig `json:"files"`
}

type jsonTaskConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type jsonTaskConfig struct {
	Name     string               `json:"name"`
	Enabled  bool                 `json:"enabled"`
	Interval string               `json:"interval"`
	Config   []jsonTaskConfigItem `json:"config"`
}

type jsonTasksConfig struct {
	Tasks []jsonTaskConfig `json:"tasks"`
}

type JsonConfig struct {
	ForceStaticConfig bool                 `json:"forceStaticConfig"`
	Host              *jsonHostConfig      `json:"host"`
	Forwarder         *jsonForwarderConfig `json:"forwarder"`
	Recipient         *jsonRecipientConfig `json:"recipient"`
	Sqlite            *jsonSqliteConfig    `json:"sqlite"`
	Web               *jsonWebConfig       `json:"web"`

	Files     []jsonFileConfig     `json:"files"`
	FileTypes []jsonFileTypeConfig `json:"fileTypes"`
	HostTypes []jsonHostTypeConfig `json:"hostTypes"`
	Tasks     jsonTasksConfig      `json:"tasks"`
}

var defaultConfig = Config{
	ForceStaticConfig: false,
	Forwarder: &ForwarderConfig{
		Enabled:            false,
		MaxBufferedEvents:  1000000,
		RecipientAddress:   "http://localhost:8081",
		ConfigPollInterval: 1 * time.Minute,
	},

	Recipient: &RecipientConfig{
		Enabled: false,
		Address: ":8081",
	},

	SQLite: &SqliteConfig{
		DatabaseFile: "logsuck.db",
		TrueBatch:    true,
	},

	Web: &WebConfig{
		Enabled:          true,
		Address:          ":8080",
		UsePackagedFiles: true,
	},
}

func FromJSON(cfg JsonConfig) (*Config, error) {
	var err error
	var hostName string
	var hostType string
	if cfg.Host != nil {
		if cfg.Host.Name == "" {
			hostName, err = getDefaultHostName()
			if err != nil {
				return nil, err
			}
		} else {
			hostName = cfg.Host.Name
		}
		if cfg.Host.Type == "" {
			hostType = cfg.Host.Type
		}
	} else {
		hostName, err = getDefaultHostName()
		if err != nil {
			return nil, err
		}
		hostType = "DEFAULT"
	}

	files := make(map[string]FileConfig, len(cfg.Files))
	for _, f := range cfg.Files {
		files[f.Filename] = FileConfig{
			Filename:  f.Filename,
			Filetypes: f.FileTypes,
		}
	}

	fileTypes, err := FileTypeConfigFromJSON(cfg.FileTypes)
	if err != nil {
		return nil, err
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
		if cfg.Forwarder.ConfigPollInterval == "" {
			log.Printf("using defaultConfigPollInterval=%v\n", defaultConfig.Forwarder.ConfigPollInterval)
			forwarder.ConfigPollInterval = defaultConfig.Forwarder.ConfigPollInterval
		} else {
			d, err := time.ParseDuration(cfg.Forwarder.ConfigPollInterval)
			if err != nil {
				log.Printf("failed to parse configPollInterval=%v, will use defaultConfigPollInterval=%v: %v\n", cfg.Forwarder.ConfigPollInterval, defaultConfig.Forwarder.ConfigPollInterval, err)
				forwarder.ConfigPollInterval = defaultConfig.Forwarder.ConfigPollInterval
			} else {
				forwarder.ConfigPollInterval = d
			}
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

	hostTypes := map[string]HostTypeConfig{}
	for _, v := range cfg.HostTypes {
		files := make([]HostFileConfig, 0, len(v.Files))
		for _, f := range v.Files {
			files = append(files, HostFileConfig{
				Name: f.FileName,
			})
		}
		hostTypes[v.Name] = HostTypeConfig{
			Files: files,
		}
	}

	tasksConfig := TasksConfig{
		Tasks: map[string]TaskConfig{},
	}
	for _, v := range cfg.Tasks.Tasks {
		enabled := v.Enabled
		intervalDuration, err := time.ParseDuration(v.Interval)
		if err != nil {
			log.Printf("got invalid duration=%v when parsing interval for task with name=%v. This task will be disabled. error: %v\n", v.Interval, v.Name, err)
			enabled = false
		}
		cfgMap := map[string]any{}
		for _, kv := range v.Config {
			cfgMap[kv.Key] = kv.Value
		}
		tasksConfig.Tasks[v.Name] = TaskConfig{
			Name:     v.Name,
			Enabled:  enabled,
			Interval: intervalDuration,
			Config:   cfgMap,
		}
	}

	return &Config{
		HostName:          hostName,
		HostType:          hostType,
		ForceStaticConfig: cfg.ForceStaticConfig,

		Forwarder: forwarder,
		Recipient: recipient,

		SQLite: sqlite,

		Web: web,

		Files:     files,
		FileTypes: fileTypes,
		HostTypes: hostTypes,
		Tasks:     tasksConfig,
	}, nil
}

func getDefaultHostName() (string, error) {
	log.Println("No hostName in configuration, will try to get host name from operating system.")
	hostName, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("error getting host name: %w", err)
	}
	log.Printf("Got host name from operating system. hostName=%v\n", hostName)
	return hostName, nil
}

func ToJSON(c *Config) (*JsonConfig, error) {
	files := make([]jsonFileConfig, 0, len(c.Files))
	for _, v := range c.Files {
		files = append(files, jsonFileConfig{
			Filename:  v.Filename,
			FileTypes: v.Filetypes,
		})
	}
	fileTypes := make([]jsonFileTypeConfig, 0, len(c.FileTypes))
	for _, v := range c.FileTypes {
		parserType := ""
		if v.ParserType == ParserTypeRegex {
			parserType = "Regex"
		} else {
			return nil, fmt.Errorf("failed to convert config to json: unknown parserType=%v", v.ParserType)
		}
		fieldExtractors := make([]string, len(v.Regex.FieldExtractors))
		for i, fe := range v.Regex.FieldExtractors {
			fieldExtractors[i] = fe.String()
		}
		fileTypes = append(fileTypes, jsonFileTypeConfig{
			Name:         v.Name,
			TimeLayout:   v.TimeLayout,
			ReadInterval: v.ReadInterval.String(),
			Parser: &jsonFileTypeParserConfig{
				Type: parserType,
				RegexConfig: &jsonRegexFileTypeParserConfig{
					EventDelimiter:  v.Regex.EventDelimiter.String(),
					FieldExtractors: fieldExtractors,
				},
			},
		})
	}
	hostTypes := make([]jsonHostTypeConfig, 0, len(c.HostTypes))
	for k, v := range c.HostTypes {
		hostTypeFileConfigs := make([]jsonHostTypeFileConfig, len(v.Files))
		for i, f := range v.Files {
			hostTypeFileConfigs[i] = jsonHostTypeFileConfig{
				FileName: f.Name,
			}
		}
		hostTypes = append(hostTypes, jsonHostTypeConfig{
			Name:  k,
			Files: hostTypeFileConfigs,
		})
	}
	tasks := make([]jsonTaskConfig, 0, len(c.Tasks.Tasks))
	for _, t := range c.Tasks.Tasks {
		cfgArray := make([]jsonTaskConfigItem, 0, len(t.Config))
		for k, v := range t.Config {
			cfgArray = append(cfgArray, jsonTaskConfigItem{
				Key:   k,
				Value: v.(string),
			})
		}
		tasks = append(tasks, jsonTaskConfig{
			Name:     t.Name,
			Enabled:  t.Enabled,
			Interval: t.Interval.String(),
			Config:   cfgArray,
		})
	}
	jsonCfg := JsonConfig{
		ForceStaticConfig: c.ForceStaticConfig,
		Host: &jsonHostConfig{
			Name: c.HostName,
			Type: c.HostType,
		},
		Forwarder: &jsonForwarderConfig{
			Enabled:            &c.Forwarder.Enabled,
			MaxBufferedEvents:  &c.Forwarder.MaxBufferedEvents,
			RecipientAddress:   c.Forwarder.RecipientAddress,
			ConfigPollInterval: c.Forwarder.ConfigPollInterval.String(),
		},
		Recipient: &jsonRecipientConfig{
			Enabled: &c.Recipient.Enabled,
			Address: c.Recipient.Address,
		},
		Sqlite: &jsonSqliteConfig{
			FileName:  c.SQLite.DatabaseFile,
			TrueBatch: &c.SQLite.TrueBatch,
		},
		Web: &jsonWebConfig{
			Enabled:          &c.Web.Enabled,
			Address:          c.Web.Address,
			UsePackagedFiles: &c.Web.UsePackagedFiles,
			DebugMode:        &c.Web.DebugMode,
		},
		Files:     files,
		FileTypes: fileTypes,
		HostTypes: hostTypes,
		Tasks: jsonTasksConfig{
			Tasks: tasks,
		},
	}
	return &jsonCfg, nil
}
