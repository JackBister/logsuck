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

package indexedfiles

import (
	"log"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
)

// IndexedFileConfig contains configuration for a specific file which will be indexed
type IndexedFileConfig struct {
	// Filename is the name of the file. It can also be a glob pattern. If the glob pattern matches multiple files, multiple watchers will be started.
	Filename   string
	FileParser parser.FileParser
	// ReadInterval is the time the file watcher will sleep between looking for new events in the file.
	// A lower duration will make events arrive faster in the search engine, but will consume more CPU.
	// The default is 1 * time.Second.
	ReadInterval time.Duration
	// TimeLayout is the layout of the _time field if it is extracted, following Go's time.Parse style https://golang.org/pkg/time/#Parse
	// The default is "2006/01/02 15:04:05"
	TimeLayout string
}

var defaultReadInterval = 1 * time.Second
var defaultTimeLayout = "2006/01/02 15:04:05"

func mergeConfigs(filename string, fileTypes []config.FileTypeConfig) (*IndexedFileConfig, error) {
	var regexParserConfig *parser.RegexParserConfig
	timeLayout := ""
	var readInterval *time.Duration
	for _, t := range fileTypes {
		if timeLayout == "" {
			timeLayout = t.TimeLayout
		} else {
			if t.Name != "DEFAULT" && timeLayout != t.TimeLayout {
				log.Printf("encountered multiple timeLayouts for filename=%v. will use timeLayout=%v. to avoid this error all fileTypes used by this filename must have the same timeLayout.\n", filename, timeLayout)
			}
		}

		if readInterval == nil {
			readInterval = &t.ReadInterval
		} else {
			if t.Name != "DEFAULT" && readInterval != &t.ReadInterval {
				log.Printf("encountered multiple readIntervals for filename=%v. will use readInterval=%v. to avoid this error all fileTypes used by this filename must have the same readInterval.\n", filename, readInterval)
			}
		}

		if t.Regex != nil {
			if regexParserConfig == nil {
				regexParserConfig = &parser.RegexParserConfig{}
			}
			eventDelimiter := t.Regex.EventDelimiter
			if regexParserConfig.EventDelimiter == nil {
				regexParserConfig.EventDelimiter = eventDelimiter
			} else {
				if t.Name != "DEFAULT" && regexParserConfig.EventDelimiter.String() != eventDelimiter.String() {
					log.Printf("encountered multiple eventDelimiters for filename=%v. will use eventDelimiter=%v. to avoid this error all fileTypes used by this filename must have the same eventDelimiter in their regex config.\n", filename, eventDelimiter)
				}
			}
			regexParserConfig.FieldExtractors = append(regexParserConfig.FieldExtractors, t.Regex.FieldExtractors...)
		} else {
			log.Printf("unhandled parser type=%v for filename=%v\n", t.ParserType, filename)
		}
	}

	if timeLayout == "" {
		timeLayout = defaultTimeLayout
	}

	if readInterval == nil {
		readInterval = &defaultReadInterval
	}

	var fp parser.FileParser
	if regexParserConfig != nil {
		fp = &parser.RegexFileParser{
			Cfg: *regexParserConfig,
		}
	}

	return &IndexedFileConfig{
		Filename:     filename,
		ReadInterval: *readInterval,
		TimeLayout:   timeLayout,
		FileParser:   fp,
	}, nil
}

func ReadFileConfig(cfg *config.Config) ([]IndexedFileConfig, error) {
	fileTypes := cfg.FileTypes
	files := cfg.Files
	indexedFiles := make([]IndexedFileConfig, 0, len(fileTypes))
	hostType := cfg.HostTypes[cfg.HostType]
	defaultHostType := cfg.HostTypes["DEFAULT"]
	hostTypeFiles := append(hostType.Files, defaultHostType.Files...)
	for _, v := range hostTypeFiles {
		fileCfg, ok := files[v.Name]
		if !ok {
			log.Printf("Failed to find config for file with filename=%v. This filename will be ignored.\n", v.Name)
			continue
		}
		fileTypeCfgs := make([]config.FileTypeConfig, 0, len(fileCfg.Filetypes))
		for _, ftn := range append(fileCfg.Filetypes, "DEFAULT") {
			ftc, ok := fileTypes[ftn]
			if !ok {
				log.Printf("Failed to find fileType with name=%v when configuring filename=%v. This file will be indexed but may be incorrectly configured.\n", ftn, v.Name)
				continue
			}
			fileTypeCfgs = append(fileTypeCfgs, ftc)
		}
		ifc, err := mergeConfigs(v.Name, fileTypeCfgs)
		if err != nil {
			log.Printf("Failed to merge configuration for file with filename=%v. This filename will be ignored. error: %v\n", v.Name, err)
		}
		indexedFiles = append(indexedFiles, *ifc)
	}
	return indexedFiles, nil
}
