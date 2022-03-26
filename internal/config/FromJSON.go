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
	"time"
)

type jsonHostConfig struct {
	Name string `json:name`
	Type string `json:type`
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

type jsonWebConfig struct {
	Enabled          *bool  `json:enabled`
	Address          string `json:address`
	UsePackagedFiles *bool  `json:usePackagedFiles`
	DebugMode        *bool  `json:debugMode`
}

type jsonConfig struct {
	ConfigPollInterval string               `json:configPollInterval`
	Host               *jsonHostConfig      `json:host`
	Forwarder          *jsonForwarderConfig `json:forwarder`
	Recipient          *jsonRecipientConfig `json:recipient`
	Sqlite             *jsonSqliteConfig    `json:sqlite`
	Web                *jsonWebConfig       `json:web`
}

var defaultConfig = StaticConfig{
	ConfigPollInterval: 1 * time.Minute,
	Forwarder: &ForwarderConfig{
		Enabled:           false,
		MaxBufferedEvents: 1000000,
		RecipientAddress:  "http://localhost:8081",
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

func FromJSON(r io.Reader) (*StaticConfig, error) {
	var cfg jsonConfig
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config JSON: %w", err)
	}

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

	var configPollInterval time.Duration
	if cfg.ConfigPollInterval != "" {
		d, err := time.ParseDuration(cfg.ConfigPollInterval)
		if err != nil {
			log.Printf("failed to parse configPollInterval=%v, will use defaultConfigPollInterval=%v: %v\n", cfg.ConfigPollInterval, defaultConfig.ConfigPollInterval, err)
			configPollInterval = defaultConfig.ConfigPollInterval
		} else {
			configPollInterval = d
		}
	} else {
		log.Printf("using defaultConfigPollInterval=%v\n", defaultConfig.ConfigPollInterval)
		configPollInterval = defaultConfig.ConfigPollInterval
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

	return &StaticConfig{
		HostName:           hostName,
		HostType:           hostType,
		ConfigPollInterval: configPollInterval,

		Forwarder: forwarder,
		Recipient: recipient,

		SQLite: sqlite,

		Web: web,
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
