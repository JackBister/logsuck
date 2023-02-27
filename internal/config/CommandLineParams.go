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
	"os"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/parser"
	"go.uber.org/zap"
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
	flag.BoolVar(&ret.PrintVersion, "version", false, "Print version info and quit.")
	flag.StringVar(&ret.Recipient, "recipient", "", "Enables recipient mode and sets the port to expose the recipient on. Recipient mode is off by default.")
	flag.StringVar(&ret.TimeField, "timefield", "_time", "The name of the field which will contain the timestamp of the event. Default '_time'.")
	flag.StringVar(&ret.TimeLayout, "timelayout", "2006/01/02 15:04:05", "The layout of the timestamp which will be extracted in the time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants. There are also the special timelayouts \"UNIX\", \"UNIX_MILLIS\", and \"UNIX_DECIMAL_NANOS\". \"UNIX\" expects the _time field to contain the number of seconds since the Unix epoch, \"UNIX_MILLIS\" expects it to contain the number of milliseconds since the Unix epoch, and UNIX_DECIMAL_NANOS expects it to contain a string of the form \"<UNIX>.<NANOS>\" where \"<UNIX>\" is the number of seconds since the Unix epoch and \"<NANOS>\" is the number of elapsed nanoseconds in that second.")
	flag.StringVar(&ret.WebAddr, "webaddr", ":8080", "The address on which the search GUI will be exposed.")
	flag.Parse()
	return &ret
}

func (c *CommandLineFlags) ToConfig(logger *zap.Logger) *Config {
	staticConfig := Config{
		HostType: "DEFAULT",

		Forwarder: &ForwarderConfig{
			Enabled:            false,
			MaxBufferedEvents:  5000,
			ConfigPollInterval: 1 * time.Minute,
		},

		Recipient: &RecipientConfig{
			Enabled: false,
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
	cfgFile, err := os.Open(c.CfgFile)
	if err == nil {
		var jsonCfg JsonConfig
		err = json.NewDecoder(cfgFile).Decode(&jsonCfg)
		if err != nil {
			logger.Fatal("error decoding json from config file", zap.String("fileName", c.CfgFile), zap.Error(err))
		}
		newCfg, err := FromJSON(jsonCfg, logger.Named("configFromJSON"))
		if err != nil {
			logger.Fatal("error parsing configuration from config file", zap.String("fileName", c.CfgFile), zap.Error(err))
		}
		staticConfig = *newCfg
		logger.Info("using configuration from file", zap.String("fileName", c.CfgFile), zap.Any("staticConfig", staticConfig))
	} else {
		logger.Warn("Could not open config file, will use command line configuration", zap.String("fileName", c.CfgFile))
		if c.HostName != "" {
			staticConfig.HostName = c.HostName
		} else {
			hostName, err := os.Hostname()
			if err != nil {
				logger.Fatal("error getting hostname", zap.Error(err))
			}
			staticConfig.HostName = hostName
		}

		if c.DatabaseFile != "" {
			staticConfig.SQLite.DatabaseFile = c.DatabaseFile
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

		var jsonParserConfig *parser.JsonParserConfig
		var regexParserConfig *parser.RegexParserConfig
		var parserType ParserType
		if c.JsonParser {
			parserType = ParserTypeJSON
			jsonParserConfig = &parser.JsonParserConfig{
				EventDelimiter: regexp.MustCompile(c.EventDelimiter),
				TimeField:      c.TimeField,
			}
		} else {
			parserType = ParserTypeRegex
			fieldExtractors := make([]*regexp.Regexp, len(c.FieldExtractors))
			if len(c.FieldExtractors) > 0 {
				for i, fe := range c.FieldExtractors {
					fieldExtractors[i] = regexp.MustCompile(fe)
				}
			}
			regexParserConfig = &parser.RegexParserConfig{
				EventDelimiter:  regexp.MustCompile(c.EventDelimiter),
				FieldExtractors: fieldExtractors,
				TimeField:       c.TimeField,
			}
		}

		staticConfig.FileTypes = map[string]FileTypeConfig{
			"DEFAULT": {
				Name:         "DEFAULT",
				TimeLayout:   c.TimeLayout,
				ReadInterval: 1 * time.Second,
				ParserType:   parserType,
				JSON:         jsonParserConfig,
				Regex:        regexParserConfig,
			},
		}

		files := map[string]FileConfig{}
		hostFiles := make([]HostFileConfig, len(flag.Args()))
		for i, file := range flag.Args() {
			files[file] = FileConfig{
				Filename:  file,
				Filetypes: []string{"DEFAULT"},
			}
			hostFiles[i] = HostFileConfig{Name: file}
		}
		staticConfig.Files = files
		staticConfig.HostTypes = map[string]HostTypeConfig{
			"DEFAULT": {
				Files: hostFiles,
			},
		}
	}
	return &staticConfig
}
