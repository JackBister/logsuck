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

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/forwarder"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/parser"
	"github.com/jackbister/logsuck/internal/recipient"
	"github.com/jackbister/logsuck/internal/tasks"
	"github.com/jackbister/logsuck/internal/web"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

var staticConfig = config.Config{
	HostType: "DEFAULT",

	Forwarder: &config.ForwarderConfig{
		Enabled:            false,
		MaxBufferedEvents:  5000,
		ConfigPollInterval: 1 * time.Minute,
	},

	Recipient: &config.RecipientConfig{
		Enabled: false,
	},

	SQLite: &config.SqliteConfig{
		DatabaseFile: "logsuck.db",
		TrueBatch:    true,
	},

	Web: &config.WebConfig{
		Enabled:          true,
		Address:          ":8080",
		UsePackagedFiles: true,
	},
}

var defaultFieldExtractors = []string{
	"(\\w+)=(\\w+)",
	"^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)",
}

type flagStringArray []string

func (i *flagStringArray) String() string {
	return fmt.Sprint(*i)
}

func (i *flagStringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var versionString string // This must be set using -ldflags "-X main.versionString=<version>" when building for --version to work

var cfgFileFlag string
var databaseFileFlag string
var eventDelimiterFlag string
var forceStaticConfigFlag bool
var forwarderFlag string
var fieldExtractorFlags flagStringArray
var hostNameFlag string
var jsonParserFlag bool
var logTypeFlag string
var printVersion bool
var recipientFlag string
var timeFieldFlag string
var timeLayoutFlag string
var webAddrFlag string

func main() {
	flag.StringVar(&cfgFileFlag, "config", "logsuck.json", "The name of the file containing the configuration for Logsuck. If a config file exists, all other command line configuration will be ignored.")
	flag.StringVar(&databaseFileFlag, "dbfile", "logsuck.db", "The name of the file in which Logsuck will store its data. If the name ':memory:' is used, no file will be created and everything will be stored in memory. If the file does not exist, a new file will be created.")
	flag.StringVar(&eventDelimiterFlag, "delimiter", "\n", "The delimiter between events in the log. Usually \\n.")
	flag.Var(&fieldExtractorFlags, "fieldextractor",
		"A regular expression which will be used to extract field values from events.\n"+
			"Can be given in two variants:\n"+
			"1. An expression containing any number of named capture groups. The names of the capture groups will be used as the field names and the captured strings will be used as the values.\n"+
			"2. An expression with two unnamed capture groups. The first capture group will be used as the field name and the second group as the value.\n"+
			"If a field with the name '_time' is extracted and matches the given timelayout, it will be used as the timestamp of the event. Otherwise the time the event was read will be used.\n"+
			"Multiple extractors can be specified by using the fieldextractor flag multiple times. "+
			"(defaults \"(\\w+)=(\\w+)\" and \"(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)\")")
	flag.BoolVar(&forceStaticConfigFlag, "forceStaticConfig", false, "If enabled, the JSON configuration file will be used instead of the configuration saved in the database. This means that you cannot alter configuration at runtime and must instead update the JSON file and restart logsuck. Has no effect in forwarder mode. Default false.")
	flag.StringVar(&forwarderFlag, "forwarder", "", "Enables forwarder mode and sets the address to forward events to. Forwarding is off by default.")
	flag.StringVar(&hostNameFlag, "hostname", "", "The name of the host running this instance of logsuck. By default, logsuck will attempt to retrieve the hostname from the operating system.")
	flag.BoolVar(&jsonParserFlag, "json", false, "Parse the given files as JSON instead of using Regex to parse. The fieldexctractor flag will be ignored. Disabled by default.")
	flag.StringVar(&logTypeFlag, "logType", "production", "The type of logger to use. Set it to 'development' to get human readable logging instead of JSON logging")
	flag.BoolVar(&printVersion, "version", false, "Print version info and quit.")
	flag.StringVar(&recipientFlag, "recipient", "", "Enables recipient mode and sets the port to expose the recipient on. Recipient mode is off by default.")
	flag.StringVar(&timeFieldFlag, "timefield", "_time", "The name of the field which will contain the timestamp of the event. Default '_time'.")
	flag.StringVar(&timeLayoutFlag, "timelayout", "2006/01/02 15:04:05", "The layout of the timestamp which will be extracted in the time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants. There are also the special timelayouts \"UNIX\", \"UNIX_MILLIS\", and \"UNIX_DECIMAL_NANOS\". \"UNIX\" expects the _time field to contain the number of seconds since the Unix epoch, \"UNIX_MILLIS\" expects it to contain the number of milliseconds since the Unix epoch, and UNIX_DECIMAL_NANOS expects it to contain a string of the form \"<UNIX>.<NANOS>\" where \"<UNIX>\" is the number of seconds since the Unix epoch and \"<NANOS>\" is the number of elapsed nanoseconds in that second.")
	flag.StringVar(&webAddrFlag, "webaddr", ":8080", "The address on which the search GUI will be exposed.")
	flag.Parse()
	if len(fieldExtractorFlags) == 0 {
		fieldExtractorFlags = defaultFieldExtractors
	}

	if printVersion {
		if versionString == "" {
			fmt.Println("(unknown version)")
			return
		}
		fmt.Println(versionString)
		return
	}

	var err error
	var logger *zap.Logger
	if logTypeFlag == "development" {
		logger, err = zap.NewDevelopment()

	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed to create Zap logger: %v", err)
		return
	}

	cfgFile, err := os.Open(cfgFileFlag)
	if err == nil {
		var jsonCfg config.JsonConfig
		err = json.NewDecoder(cfgFile).Decode(&jsonCfg)
		if err != nil {
			logger.Fatal("error decoding json from config file", zap.String("fileName", cfgFileFlag), zap.Error(err))
			return
		}
		newCfg, err := config.FromJSON(jsonCfg, logger.Named("configFromJSON"))
		if err != nil {
			logger.Fatal("error parsing configuration from config file", zap.String("fileName", cfgFileFlag), zap.Error(err))
			return
		}
		staticConfig = *newCfg
		logger.Info("using configuration from file", zap.String("fileName", cfgFileFlag), zap.Any("staticConfig", staticConfig))
	} else {
		logger.Warn("Could not open config file, will use command line configuration", zap.String("fileName", cfgFileFlag))
		if hostNameFlag != "" {
			staticConfig.HostName = hostNameFlag
		} else {
			hostName, err := os.Hostname()
			if err != nil {
				logger.Fatal("error getting hostname", zap.Error(err))
				return
			}
			staticConfig.HostName = hostName
		}

		if databaseFileFlag != "" {
			staticConfig.SQLite.DatabaseFile = databaseFileFlag
		}
		if webAddrFlag != "" {
			staticConfig.Web.Address = webAddrFlag
		}
		if forwarderFlag != "" {
			staticConfig.Forwarder.Enabled = true
			staticConfig.Forwarder.RecipientAddress = forwarderFlag
			staticConfig.Web.Enabled = false
		}
		if recipientFlag != "" {
			staticConfig.Recipient.Enabled = true
			staticConfig.Recipient.Address = recipientFlag
		}

		var jsonParserConfig *parser.JsonParserConfig
		var regexParserConfig *parser.RegexParserConfig
		var parserType config.ParserType
		if jsonParserFlag {
			parserType = config.ParserTypeJSON
			jsonParserConfig = &parser.JsonParserConfig{
				EventDelimiter: regexp.MustCompile(eventDelimiterFlag),
				TimeField:      timeFieldFlag,
			}
		} else {
			parserType = config.ParserTypeRegex
			fieldExtractors := make([]*regexp.Regexp, len(fieldExtractorFlags))
			if len(fieldExtractorFlags) > 0 {
				for i, fe := range fieldExtractorFlags {
					fieldExtractors[i] = regexp.MustCompile(fe)
				}
			}
			regexParserConfig = &parser.RegexParserConfig{
				EventDelimiter:  regexp.MustCompile(eventDelimiterFlag),
				FieldExtractors: fieldExtractors,
				TimeField:       timeFieldFlag,
			}
		}

		staticConfig.FileTypes = map[string]config.FileTypeConfig{
			"DEFAULT": {
				Name:         "DEFAULT",
				TimeLayout:   timeLayoutFlag,
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

	ctx := context.Background()

	var configRepo config.ConfigRepository
	var configSource config.ConfigSource
	var jobRepo jobs.Repository
	var jobEngine *jobs.Engine
	var publisher events.EventPublisher
	var repo events.Repository
	if staticConfig.Forwarder.Enabled {
		publisher = forwarder.ForwardingEventPublisher(&staticConfig, logger.Named("ForwardingEventPublisher"))
		configSource = forwarder.NewRemoteConfigSource(&staticConfig, logger.Named("RemoteConfigSource"))
	} else {
		additionalSqliteParameters := "?_journal_mode=WAL"
		if staticConfig.SQLite.DatabaseFile == ":memory:" {
			// cache=shared breaks DeleteOldEventsTask. But not having it breaks everything in :memory: mode.
			// So we set cache=shared for :memory: mode and assume people will not need to delete old tasks in that mode.
			additionalSqliteParameters += "&cache=shared"
		}
		db, err := sql.Open("sqlite3", "file:"+staticConfig.SQLite.DatabaseFile+additionalSqliteParameters)
		if err != nil {
			logger.Fatal("error when creating sqlite database", zap.Error(err))
			return
		}
		configRepo, err = config.NewSqliteConfigRepository(&staticConfig, db, !forceStaticConfigFlag && !staticConfig.ForceStaticConfig, logger.Named("SqliteConfigRepository"))
		if err != nil {
			logger.Fatal("error when creating sqlite config repository", zap.Error(err))
			return
		}
		repo, err = events.SqliteRepository(db, staticConfig.SQLite, logger.Named("SqliteEventsRepository"))
		if err != nil {
			logger.Fatal("error when creating sqlite event repository", zap.Error(err))
			return
		}
		jobRepo, err = jobs.SqliteRepository(db)
		if err != nil {
			logger.Fatal("error when creating sqlite job repository", zap.Error(err))
			return
		}
		publisher = events.BatchedRepositoryPublisher(&staticConfig, repo, logger.Named("BatchedRepositoryPublisher"))
		if forceStaticConfigFlag || staticConfig.ForceStaticConfig {
			logger.Info("Static configuration is forced. Configuration will not be saved to database and will only be read from the JSON configuration file. Remove the forceStaticConfig flag from the command line or configuration file in order to use dynamic configuration.")
			configSource = &config.StaticConfigSource{
				Config: staticConfig,
			}
		} else {
			configSource = configRepo
		}
	}
	dynamicConfig, err := configSource.Get()
	if err != nil {
		logger.Fatal("failed to get dynamic configuration during init", zap.Error(err))
		return
	}

	jobEngine = jobs.NewEngine(configSource, repo, jobRepo, logger.Named("JobEngine"))
	indexedFiles, err := indexedfiles.ReadFileConfig(&dynamicConfig.Cfg, logger)
	if err != nil {
		logger.Fatal("got error when reading dynamic file config", zap.Error(err))
		return
	}

	watchers := map[string]*files.GlobWatcher{}
	if staticConfig.Recipient.Enabled {
		logger.Info("recipient is enabled, will not read any files on this host.")
	} else {
		reloadFileWatchers(logger.Named("reloadFileWatchers"), &watchers, indexedFiles, &dynamicConfig.Cfg, publisher, ctx)
	}

	if staticConfig.Recipient.Enabled {
		go func() {
			logger.Fatal("got error from recipient Serve method", zap.Error(recipient.NewRecipientEndpoint(configSource, repo, logger.Named("RecipientEndpoint")).Serve()))
		}()
	}

	if staticConfig.Web.Enabled {
		go func() {
			logger.Fatal("got error from web Serve method", zap.Error(web.NewWeb(configSource, configRepo, repo, jobRepo, jobEngine, logger.Named("Web")).Serve()))
		}()
	}

	tm := tasks.NewTaskManager(
		&dynamicConfig.Cfg.Tasks, tasks.TaskContext{
			EventsRepo: repo,
		},
		ctx,
		logger.Named("TaskManager"))
	tm.UpdateConfig(dynamicConfig.Cfg)

	go func() {
		for {
			<-configSource.Changes()
			newCfg, err := configSource.Get()
			if err != nil {
				logger.Warn("got error when reading updated dynamic file config. file and task config will not be updated", zap.Error(err))
				continue
			}
			if !staticConfig.Recipient.Enabled {
				newIndexedFiles, err := indexedfiles.ReadFileConfig(&newCfg.Cfg, logger)
				if err != nil {
					logger.Warn("got error when reading updated dynamic file config. file config will not be updated", zap.Error(err))
				} else {
					reloadFileWatchers(logger.Named("reloadFileWatchers"), &watchers, newIndexedFiles, &newCfg.Cfg, publisher, ctx)
				}
			}
			tm.UpdateConfig(newCfg.Cfg)
		}
	}()

	select {}
}

func reloadFileWatchers(logger *zap.Logger, watchers *map[string]*files.GlobWatcher, indexedFiles []indexedfiles.IndexedFileConfig, cfg *config.Config, publisher events.EventPublisher, ctx context.Context) {
	logger.Info("reloading file watchers", zap.Int("newIndexedFilesLen", len(indexedFiles)), zap.Int("oldIndexedFilesLen", len(*watchers)))
	indexedFilesMap := map[string]indexedfiles.IndexedFileConfig{}
	for _, cfg := range indexedFiles {
		indexedFilesMap[cfg.Filename] = cfg
	}
	watchersToDelete := []string{}
	// Update existing watchers and find watchers to delete
	for k, v := range *watchers {
		newCfg, ok := indexedFilesMap[k]
		if !ok {
			logger.Info("filename not found in new indexed files config. will cancel and delete watcher", zap.String("fileName", k))
			v.Cancel()
			watchersToDelete = append(watchersToDelete, k)
			continue
		}
		v.UpdateConfig(newCfg)
	}

	// delete watchers that do not exist in the new config
	for _, k := range watchersToDelete {
		delete(*watchers, k)
	}

	// Add new watchers
	for k, v := range indexedFilesMap {
		_, ok := (*watchers)[k]
		if ok {
			continue
		}
		logger.Info("creating new watcher", zap.String("fileName", k))
		w, err := files.NewGlobWatcher(v, v.Filename, staticConfig.HostName, publisher, ctx, logger.Named("GlobWatcher"))
		if err != nil {
			logger.Warn("got error when creating GlobWatcher", zap.String("fileName", v.Filename), zap.Error(err))
			continue
		}
		(*watchers)[k] = w
	}
}
