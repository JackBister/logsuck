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

package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/plugins/sqlite_common"
)

type FlagStringArray []string

func (i *FlagStringArray) String() string {
	return fmt.Sprint(*i)
}

func (i *FlagStringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type CommandLineFlags struct {
	CfgFile           string
	DatabaseFile      string
	EventDelimiter    string
	ForceStaticConfig bool
	Forwarder         string
	FieldExtractors   FlagStringArray
	HostName          string
	JsonParser        bool
	LogType           string
	PrintJsonSchema   bool
	PrintVersion      bool
	Recipient         string
	TimeField         string
	TimeLayout        string
	WebAddr           string
}

func ParseCommandLine() *CommandLineFlags {
	ret := CommandLineFlags{}
	flag.StringVar(&ret.CfgFile, "config", "logsuck.json", "The name of the file containing the configuration for Logsuck. If a config file exists, all other command line configuration will be ignored.")
	flag.StringVar(&ret.DatabaseFile, "dbfile", "logsuck.db", "The name of the file in which Logsuck will store its data. If the name ':memory:' is used, no file will be created and everything will be stored in memory. If the file does not exist, a new file will be created.")
	flag.StringVar(&ret.EventDelimiter, "delimiter", "\n", "The delimiter between events in the log. Usually \\n.")
	flag.Var(&ret.FieldExtractors, "fieldextractor",
		"A regular expression which will be used to extract field values from events.\n"+
			"Can be given in two variants:\n"+
			"1. An expression containing any number of named capture groups. The names of the capture groups will be used as the field names and the captured strings will be used as the values.\n"+
			"2. An expression with two unnamed capture groups. The first capture group will be used as the field name and the second group as the value.\n"+
			"If a field with the name '_time' is extracted and matches the given timelayout, it will be used as the timestamp of the event. Otherwise the time the event was read will be used.\n"+
			"Multiple extractors can be specified by using the fieldextractor flag multiple times. "+
			"(defaults \"(\\w+)=(\\w+)\" and \"(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)\")")
	flag.BoolVar(&ret.ForceStaticConfig, "forceStaticConfig", false, "If enabled, the JSON configuration file will be used instead of the configuration saved in the database. This means that you cannot alter configuration at runtime and must instead update the JSON file and restart logsuck. Has no effect in forwarder mode. Default false.")
	flag.StringVar(&ret.Forwarder, "forwarder", "", "Enables forwarder mode and sets the address to forward events to. Forwarding is off by default.")
	flag.StringVar(&ret.HostName, "hostname", "", "The name of the host running this instance of logsuck. By default, logsuck will attempt to retrieve the hostname from the operating system.")
	flag.BoolVar(&ret.JsonParser, "json", false, "Parse the given files as JSON instead of using Regex to parse. The fieldexctractor flag will be ignored. Disabled by default.")
	flag.StringVar(&ret.LogType, "logType", "production", "The type of logger to use. Set it to 'development' to get human readable logging instead of JSON logging")
	flag.BoolVar(&ret.PrintJsonSchema, "schema", false, "Print configuration schema and quit.")
	flag.BoolVar(&ret.PrintVersion, "version", false, "Print version info and quit.")
	flag.StringVar(&ret.Recipient, "recipient", "", "Enables recipient mode and sets the port to expose the recipient on. Recipient mode is off by default.")
	flag.StringVar(&ret.TimeField, "timefield", "_time", "The name of the field which will contain the timestamp of the event. Default '_time'.")
	flag.StringVar(&ret.TimeLayout, "timelayout", "2006/01/02 15:04:05", "The layout of the timestamp which will be extracted in the time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants. There are also the special timelayouts \"UNIX\", \"UNIX_MILLIS\", and \"UNIX_DECIMAL_NANOS\". \"UNIX\" expects the _time field to contain the number of seconds since the Unix epoch, \"UNIX_MILLIS\" expects it to contain the number of milliseconds since the Unix epoch, and UNIX_DECIMAL_NANOS expects it to contain a string of the form \"<UNIX>.<NANOS>\" where \"<UNIX>\" is the number of seconds since the Unix epoch and \"<NANOS>\" is the number of elapsed nanoseconds in that second.")
	flag.StringVar(&ret.WebAddr, "webaddr", ":8080", "The address on which the search GUI will be exposed.")
	flag.Parse()
	return &ret
}

func (c *CommandLineFlags) ToConfig(logger *slog.Logger) *config.Config {
	staticConfig := config.Config{
		HostType: "DEFAULT",

		Forwarder: &config.ForwarderConfig{
			Enabled:            false,
			MaxBufferedEvents:  5000,
			ConfigPollInterval: 1 * time.Minute,
		},

		Recipient: &config.RecipientConfig{
			Enabled: false,
		},

		Web: &config.WebConfig{
			Enabled:          true,
			Address:          ":8080",
			UsePackagedFiles: true,
		},

		Plugins: map[string]any{
			sqlite_common.Plugin.Name: map[string]any{
				"fileName": "logsuck.db",
			},
		},
	}
	cfgFile, err := os.Open(c.CfgFile)
	if err == nil {
		var jsonCfg config.JsonConfig
		err = json.NewDecoder(cfgFile).Decode(&jsonCfg)
		if err != nil {
			logger.Error("error decoding json from config file", slog.String("fileName", c.CfgFile), slog.Any("error", err))
			os.Exit(1)
		}
		newCfg, err := config.FromJSON(jsonCfg, logger)
		if err != nil {
			logger.Error("error parsing configuration from config file", slog.String("fileName", c.CfgFile), slog.Any("error", err))
			os.Exit(1)
		}
		staticConfig = *newCfg
		logger.Info("using configuration from file", slog.String("fileName", c.CfgFile), slog.Any("staticConfig", staticConfig))
	} else {
		logger.Warn("Could not open config file, will use command line configuration", slog.String("fileName", c.CfgFile))
		if c.HostName != "" {
			staticConfig.HostName = c.HostName
		} else {
			hostName, err := os.Hostname()
			if err != nil {
				logger.Error("error getting hostname", slog.Any("error", err))
				os.Exit(1)
			}
			staticConfig.HostName = hostName
		}

		if c.DatabaseFile != "" {
			staticConfig.Plugins[sqlite_common.Plugin.Name] = map[string]any{
				"fileName": c.DatabaseFile,
			}
		}
		if c.WebAddr != "" {
			staticConfig.Web.Address = c.WebAddr
		}
		if c.Forwarder != "" {
			staticConfig.Forwarder.Enabled = true
			staticConfig.Forwarder.RecipientAddress = c.Forwarder
			staticConfig.Web.Enabled = false
		}
		if c.Recipient != "" {
			staticConfig.Recipient.Enabled = true
			staticConfig.Recipient.Address = c.Recipient
		}

		var jsonParserConfig *config.JsonParserConfig
		var regexParserConfig *config.RegexParserConfig
		var parserType config.ParserType
		if c.JsonParser {
			parserType = config.ParserTypeJSON
			jsonParserConfig = &config.JsonParserConfig{
				EventDelimiter: regexp.MustCompile(c.EventDelimiter),
				TimeField:      c.TimeField,
			}
		} else {
			parserType = config.ParserTypeRegex
			fieldExtractors := make([]*regexp.Regexp, len(c.FieldExtractors))
			if len(c.FieldExtractors) > 0 {
				for i, fe := range c.FieldExtractors {
					fieldExtractors[i] = regexp.MustCompile(fe)
				}
			}
			regexParserConfig = &config.RegexParserConfig{
				EventDelimiter:  regexp.MustCompile(c.EventDelimiter),
				FieldExtractors: fieldExtractors,
				TimeField:       c.TimeField,
			}
		}

		staticConfig.FileTypes = map[string]config.FileTypeConfig{
			"DEFAULT": {
				Name:         "DEFAULT",
				TimeLayout:   c.TimeLayout,
				ReadInterval: 1 * time.Second,
				ParserType:   parserType,
				JSON:         jsonParserConfig,
				Regex:        regexParserConfig,
			},
		}

		files := map[string]config.FileConfig{}
		hostFiles := make([]config.HostFileConfig, len(flag.Args()))
		for i, file := range flag.Args() {
			files[file] = config.FileConfig{
				Filename:  file,
				Filetypes: []string{"DEFAULT"},
			}
			hostFiles[i] = config.HostFileConfig{Name: file}
		}
		staticConfig.Files = files
		staticConfig.HostTypes = map[string]config.HostTypeConfig{
			"DEFAULT": {
				Files: hostFiles,
			},
		}
	}
	return &staticConfig
}
