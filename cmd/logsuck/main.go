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

	_ "github.com/mattn/go-sqlite3"
)

var staticConfig = config.Config{
	HostType:           "DEFAULT",
	ConfigPollInterval: 1 * time.Minute,

	Forwarder: &config.ForwarderConfig{
		Enabled:           false,
		MaxBufferedEvents: 5000,
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
var forwarderFlag string
var fieldExtractorFlags flagStringArray
var printVersion bool
var recipientFlag string
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
	flag.StringVar(&forwarderFlag, "forwarder", "", "Enables forwarder mode and sets the address to forward events to. Forwarding is off by default.")
	flag.StringVar(&timeLayoutFlag, "timelayout", "2006/01/02 15:04:05", "The layout of the timestamp which will be extracted in the _time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants.")
	flag.BoolVar(&printVersion, "version", false, "Print version info and quit.")
	flag.StringVar(&recipientFlag, "recipient", "", "Enables recipient mode and sets the port to expose the recipient on. Recipient mode is off by default.")
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

	cfgFile, err := os.Open(cfgFileFlag)
	if err == nil {
		var jsonCfg config.JsonConfig
		err = json.NewDecoder(cfgFile).Decode(&jsonCfg)
		if err != nil {
			log.Fatalf("error decoding json from file '%v': %v\n", cfgFileFlag, err)
		}
		newCfg, err := config.FromJSON(jsonCfg)
		if err != nil {
			log.Fatalf("error parsing configuration from file '%v': %v\n", cfgFileFlag, err)
		}
		staticConfig = *newCfg
		log.Printf("Using configuration from file '%v': %v\n", cfgFileFlag, staticConfig)
	} else {
		log.Printf("Could not open config file '%v', will use command line configuration\n", cfgFileFlag)
		hostName, err := os.Hostname()
		if err != nil {
			log.Fatalf("error getting hostname: %v\n", err)
		}
		staticConfig.HostName = hostName

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

		fieldExtractors := make([]*regexp.Regexp, len(fieldExtractorFlags))
		if len(fieldExtractorFlags) > 0 {
			for i, fe := range fieldExtractorFlags {
				fieldExtractors[i] = regexp.MustCompile(fe)
			}
		}

		staticConfig.FileTypes = map[string]config.FileTypeConfig{
			"DEFAULT": {
				Name:         "DEFAULT",
				TimeLayout:   timeLayoutFlag,
				ReadInterval: 1 * time.Second,
				ParserType:   config.ParserTypeRegex,
				Regex: &parser.RegexParserConfig{
					EventDelimiter:  regexp.MustCompile(eventDelimiterFlag),
					FieldExtractors: fieldExtractors,
				},
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
		publisher = forwarder.ForwardingEventPublisher(&staticConfig)
		configSource = forwarder.NewRemoteConfigSource(&staticConfig)
	} else {
		db, err := sql.Open("sqlite3", "file:"+staticConfig.SQLite.DatabaseFile+"?cache=shared&_journal_mode=WAL")
		if err != nil {
			log.Fatalln(err.Error())
		}
		configRepo, err = config.NewSqliteConfigRepository(&staticConfig, db)
		if err != nil {
			log.Fatalln(err.Error())
		}
		repo, err = events.SqliteRepository(db, staticConfig.SQLite)
		if err != nil {
			log.Fatalln(err.Error())
		}
		jobRepo, err = jobs.SqliteRepository(db)
		if err != nil {
			log.Fatalln(err.Error())
		}
		publisher = events.BatchedRepositoryPublisher(&staticConfig, repo)
		configSource = configRepo
	}
	dynamicConfig, err := configSource.Get()
	if err != nil {
		log.Fatalf("failed to get dynamic configuration during init: %v\n", err)
		return
	}

	jobEngine = jobs.NewEngine(configSource, repo, jobRepo)
	indexedFiles, err := indexedfiles.ReadFileConfig(&dynamicConfig.Cfg)
	if err != nil {
		log.Fatalln("got error when reading dynamic file config", err)
	}

	if staticConfig.Recipient.Enabled {
		log.Println("recipient is enabled, will not read any files on this host.")
	} else {
		watchers := map[string]*files.GlobWatcher{}
		reloadFileWatchers(&watchers, indexedFiles, &dynamicConfig.Cfg, publisher, ctx)

		go func() {
			for {
				<-configSource.Changes()
				newCfg, err := configSource.Get()
				if err != nil {
					log.Printf("got error when reading updated dynamic file config. file config will not be updated: %v\n", err)
					continue
				}
				newIndexedFiles, err := indexedfiles.ReadFileConfig(&newCfg.Cfg)
				if err != nil {
					log.Printf("got error when reading updated dynamic file config. file config will not be updated: %v\n", err)
					continue
				}
				reloadFileWatchers(&watchers, newIndexedFiles, &newCfg.Cfg, publisher, ctx)
			}
		}()
	}

	if staticConfig.Recipient.Enabled {
		go func() {
			log.Fatal(recipient.NewRecipientEndpoint(configSource, repo).Serve())
		}()
	}

	if staticConfig.Web.Enabled {
		go func() {
			log.Fatal(web.NewWeb(configSource, configRepo, repo, jobRepo, jobEngine).Serve())
		}()
	}

	// TODO: This should be updated on <-dynamicConfig.Changes() just like the file watchers
	tm := tasks.NewTaskManager(&dynamicConfig.Cfg.Tasks, ctx)
	err = tm.AddTask(&tasks.DeleteOldEventsTask{Repo: repo})
	if err != nil {
		log.Printf("got error when adding task: %v", err)
	}

	select {}
}

func reloadFileWatchers(watchers *map[string]*files.GlobWatcher, indexedFiles []indexedfiles.IndexedFileConfig, cfg *config.Config, publisher events.EventPublisher, ctx context.Context) {
	log.Printf("reloading file watchers. newIndexedFilesLen=%v, oldIndexedFilesLen=%v\n", len(indexedFiles), len(*watchers))
	indexedFilesMap := map[string]indexedfiles.IndexedFileConfig{}
	for _, cfg := range indexedFiles {
		indexedFilesMap[cfg.Filename] = cfg
	}
	watchersToDelete := []string{}
	// Update existing watchers and find watchers to delete
	for k, v := range *watchers {
		newCfg, ok := indexedFilesMap[k]
		if !ok {
			log.Printf("filename=%s not found in new indexed files config. will cancel and delete watcher.\n", k)
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
		log.Printf("creating new watcher for filename=%s\n", k)
		w, err := files.NewGlobWatcher(v, v.Filename, staticConfig.HostName, publisher, ctx)
		if err != nil {
			log.Printf("got error when creating GlobWatcher for filename=%s: %v", v.Filename, err)
			continue
		}
		(*watchers)[k] = w
	}
}
