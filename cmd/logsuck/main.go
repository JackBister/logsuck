// Copyright 2020 The Logsuck Authors
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

//go:generate go run ../packager/main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/jackbister/logsuck/internal/web"

	_ "github.com/mattn/go-sqlite3"
)

var cfg = config.Config{
	IndexedFiles: []config.IndexedFileConfig{},

	FieldExtractors: []*regexp.Regexp{
		regexp.MustCompile("(\\w+)=(\\w+)"),
		regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)"),
	},

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

	cfgFile, err := os.Open(cfgFileFlag)
	if err == nil {
		newCfg, err := config.FromJSON(cfgFile)
		if err != nil {
			log.Fatalf("error parsing configuration from file '%v': %v", cfgFileFlag, err)
		}
		cfg = *newCfg
		log.Printf("Using configuration from file '%v': %v\n", cfgFileFlag, cfg)
	} else {
		log.Printf("Could not open config file '%v', will use command line configuration\n", cfgFileFlag)
		hostName, err := os.Hostname()
		if err != nil {
			log.Fatalf("error getting hostname: %v\n", err)
		}
		cfg.HostName = hostName

		if databaseFileFlag != "" {
			cfg.SQLite.DatabaseFile = databaseFileFlag
		}
		if len(fieldExtractorFlags) > 0 {
			cfg.FieldExtractors = make([]*regexp.Regexp, len(fieldExtractorFlags))
			for i, fe := range fieldExtractorFlags {
				re, err := regexp.Compile(fe)
				if err != nil {
					log.Fatalf("failed to compile regex '%v': %v\n", fe, err)
				}
				cfg.FieldExtractors[i] = re
			}
		}
		if webAddrFlag != "" {
			cfg.Web.Address = webAddrFlag
		}

		cfg.IndexedFiles = make([]config.IndexedFileConfig, len(flag.Args()))
		for i, file := range flag.Args() {
			cfg.IndexedFiles[i] = config.IndexedFileConfig{
				Filename:       file,
				EventDelimiter: regexp.MustCompile(eventDelimiterFlag),
				ReadInterval:   1 * time.Second,
				TimeLayout:     timeLayoutFlag,
			}
		}
	}

	commandChannels := make([]chan files.FileWatcherCommand, len(cfg.IndexedFiles))

	var jobRepo jobs.Repository
	var jobEngine *jobs.Engine
	var publisher events.EventPublisher
	var repo events.Repository
	if cfg.Forwarder.Enabled {
		publisher = events.ForwardingEventPublisher(&cfg)
	} else {
		db, err := sql.Open("sqlite3", cfg.SQLite.DatabaseFile+"?cache=shared&_journal_mode=WAL")
		if err != nil {
			log.Fatalln(err.Error())
		}
		repo, err = events.SqliteRepository(db, cfg.SQLite)
		if err != nil {
			log.Fatalln(err.Error())
		}
		jobRepo, err = jobs.SqliteRepository(db)
		if err != nil {
			log.Fatalln(err.Error())
		}
		jobEngine = jobs.NewEngine(&cfg, repo, jobRepo)
		publisher = events.BatchedRepositoryPublisher(&cfg, repo)
	}

	// files can only be watched once. If a file is matched by multiple globs, the first one wins.
	seenFiles := map[string]struct{}{}
	for i, fileCfg := range cfg.IndexedFiles {
		globFiles, err := filepath.Glob(fileCfg.Filename)
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range globFiles {
			absfile, err := filepath.Abs(file)
			if err != nil {
				log.Fatal(err)
			}
			if _, seen := seenFiles[absfile]; seen {
				log.Printf("filename=%v was matched by glob=%v, but this file is already being watched by a previous configuration. This file will be skipped for this configuration.", absfile, fileCfg.Filename)
				continue
			}
			commandChannels[i] = make(chan files.FileWatcherCommand, 1)
			fw, err := files.NewFileWatcher(fileCfg, file, cfg.HostName, commandChannels[i], publisher)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Starting FileWatcher for filename=" + file)
			go fw.Start()
			seenFiles[absfile] = struct{}{}
		}
	}

	if cfg.Recipient.Enabled {
		go func() {
			log.Fatal(events.NewEventRecipient(&cfg, repo).Serve())
		}()
	}

	if cfg.Web.Enabled {
		go func() {
			log.Fatal(web.NewWeb(&cfg, repo, jobRepo, jobEngine).Serve())
		}()
	}

	select {}
}
