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
	"strconv"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/web"

	_ "github.com/mattn/go-sqlite3"
)

var staticConfig = config.StaticConfig{
	HostType: "DEFAULT",

	Forwarder: &config.ForwarderConfig{
		Enabled: false,
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
var fieldExtractorFlags flagStringArray
var printVersion bool
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
			"(defaults \"(\\w+)=(\\w+)\" and \"(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)\")")
	flag.StringVar(&timeLayoutFlag, "timelayout", "2006/01/02 15:04:05", "The layout of the timestamp which will be extracted in the _time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants.")
	flag.BoolVar(&printVersion, "version", false, "Print version info and quit.")
	flag.StringVar(&webAddrFlag, "webaddr", ":8080", "The address on which the search GUI will be exposed.")
	flag.Parse()

	if printVersion {
		if versionString == "" {
			fmt.Println("(unknown version)")
			return
		}
		fmt.Println(versionString)
		return
	}

	configSources := []config.ConfigSource{}
	cfgFile, err := os.Open(cfgFileFlag)
	if err == nil {
		newCfg, err := config.FromJSON(cfgFile)
		if err != nil {
			log.Fatalf("error parsing configuration from file '%v': %v", cfgFileFlag, err)
		}
		staticConfig = *newCfg
		log.Printf("Using configuration from file '%v': %v\n", cfgFileFlag, staticConfig)

		newOffset, err := cfgFile.Seek(0, 0)
		if err != nil {
			log.Printf("got error when seeking to offset 0 in json config: %v\n", err)
		} else if newOffset != 0 {
			log.Printf("got newOffset=%v when seeking to offset 0 in json config\n", newOffset)
		} else {
			var m map[string]interface{}
			decoder := json.NewDecoder(cfgFile)
			err := decoder.Decode(&m)
			if err != nil {
				log.Printf("got error when decoding json config: %v\n", err)
			} else {
				configSources = append(configSources, config.NewMapConfigSource(m))
			}
		}
	} else {
		log.Printf("Could not open config file '%v', will use command line configuration\n", cfgFileFlag)
		configMap := map[string]interface{}{}
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

		defaultFieldExtractors := make([]string, len(fieldExtractorFlags))
		if len(fieldExtractorFlags) > 0 {
			fieldExtractors := make(map[string]string, len(fieldExtractorFlags))
			for i, fe := range fieldExtractorFlags {
				k := strconv.Itoa(i)
				fieldExtractors[k] = fe
				defaultFieldExtractors[i] = k
			}
			configMap["fieldExtractors"] = fieldExtractors
		}

		defaultFileType := map[string]interface{}{}
		defaultFileType["timeLayout"] = timeLayoutFlag
		defaultFileType["parser"] = map[string]interface{}{
			"type": "Regex",
			"regexConfig": map[string]interface{}{
				"eventDelimiter":  eventDelimiterFlag,
				"fieldExtractors": defaultFieldExtractors,
			},
		}
		configMap["fileTypes"] = map[string]interface{}{
			"DEFAULT": defaultFileType,
		}

		defaultFiles := make([]map[string]interface{}, len(flag.Args()))
		for i, file := range flag.Args() {
			defaultFiles[i] = map[string]interface{}{
				"fileName":     file,
				"fileTypes":    []string{"DEFAULT"},
				"readInterval": "1s",
			}
		}
		configMap["hostTypes"] = map[string]interface{}{
			"DEFAULT": map[string]interface{}{
				"configPollInterval": "1m",
				"files":              defaultFiles,
			},
		}

		configSources = append(configSources, config.NewMapConfigSource(configMap))
	}

	ctx := context.Background()

	var jobRepo jobs.Repository
	var jobEngine *jobs.Engine
	var publisher events.EventPublisher
	var repo events.Repository
	if staticConfig.Forwarder.Enabled {
		publisher = events.ForwardingEventPublisher(&staticConfig)
	} else {
		db, err := sql.Open("sqlite3", staticConfig.SQLite.DatabaseFile+"?cache=shared&_journal_mode=WAL")
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
		sqliteConfigSource, err := config.NewSqliteConfigSource(db)
		if err != nil {
			log.Printf("got error when creating sqliteConfigSource: %v\n", err)
		} else {
			// This is a bit wack. This is because we want the sqliteConfigSource to be higher priority than jsonConfigSource
			configSources = append([]config.ConfigSource{
				config.NewPollingConfigSource(sqliteConfigSource, staticConfig.ConfigPollInterval),
			},
				configSources...)
		}
	}

	dynamicConfig := config.NewDynamicConfig(configSources)
	jobEngine = jobs.NewEngine(&staticConfig, dynamicConfig, repo, jobRepo)
	allFiles, err := indexedfiles.ReadDynamicFileConfig(dynamicConfig)
	if err != nil {
		log.Fatalln("got error when reading dynamic file config", err)
	}
	hostCfg, err := config.GetHostConfig(dynamicConfig, staticConfig.HostType)
	if err != nil {
		log.Fatalln("got error when reading host config", err)
	}
	indexedFiles := make([]indexedfiles.IndexedFileConfig, 0)
	for _, file := range hostCfg.Files {
		for _, ifc := range allFiles {
			if file.Name == ifc.Filename {
				indexedFiles = append(indexedFiles, ifc)
			}
		}
	}

	watchers := map[string]*files.GlobWatcher{}
	reloadFileWatchers(&watchers, indexedFiles, staticConfig, dynamicConfig, publisher, ctx)

	go func() {
		for {
			<-dynamicConfig.Changes()
			allFiles, err := indexedfiles.ReadDynamicFileConfig(dynamicConfig)
			if err != nil {
				log.Printf("got error when reading updated dynamic file config. file config will not be updated: %v\n", err)
				continue
			}
			newIndexedFiles := make([]indexedfiles.IndexedFileConfig, 0)
			for _, file := range hostCfg.Files {
				for _, ifc := range allFiles {
					if file.Name == ifc.Filename {
						newIndexedFiles = append(indexedFiles, ifc)
					}
				}
			}
			reloadFileWatchers(&watchers, newIndexedFiles, staticConfig, dynamicConfig, publisher, ctx)
		}
	}()

	if staticConfig.Recipient.Enabled {
		go func() {
			log.Fatal(events.NewEventRecipient(&staticConfig, repo).Serve())
		}()
	}

	if staticConfig.Web.Enabled {
		go func() {
			log.Fatal(web.NewWeb(&staticConfig, dynamicConfig, repo, jobRepo, jobEngine).Serve())
		}()
	}

	config.ReadDynamicTasksConfig(dynamicConfig)
	/*
		tm := tasks.NewTaskManager(staticConfig.Tasks, ctx)
		err = tm.AddTask(&tasks.DeleteOldEventsTask{Repo: repo})
		if err != nil {
			log.Printf("got error when adding task: %v", err)
		}
	*/

	select {}
}

func reloadFileWatchers(watchers *map[string]*files.GlobWatcher, indexedFiles []indexedfiles.IndexedFileConfig, staticConfig config.StaticConfig, dynamicConfig config.DynamicConfig, publisher events.EventPublisher, ctx context.Context) {
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
