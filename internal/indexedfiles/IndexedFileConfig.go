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
	"fmt"
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

func mergeConfigs(filename string, fileTypes []config.FileTypeConfig) (*IndexedFileConfig, error) {
	var regexParserConfig parser.RegexParserConfig
	timeLayout := ""
	var readInterval *time.Duration
	for _, t := range fileTypes {
		if timeLayout == "" {
			timeLayout = t.TimeLayout
		} else {
			if timeLayout != t.TimeLayout {
				log.Printf("encountered multiple timeLayouts for filename=%v. will use timeLayout=%v. to avoid this error all fileTypes used by this filename must have the same timeLayout.\n", filename, timeLayout)
			}
		}

		if readInterval == nil {
			readInterval = &t.ReadInterval
		} else {
			if readInterval != &t.ReadInterval {
				log.Printf("encountered multiple readIntervals for filename=%v. will use readInterval=%v. to avoid this error all fileTypes used by this filename must have the same readInterval.\n", filename, readInterval)
			}
		}

		if t.Regex != nil {
			eventDelimiter := t.Regex.EventDelimiter
			if regexParserConfig.EventDelimiter == nil {
				regexParserConfig.EventDelimiter = eventDelimiter
			} else {
				if regexParserConfig.EventDelimiter.String() != eventDelimiter.String() {
					log.Printf("encountered multiple eventDelimiters for filename=%v. will use eventDelimiter=%v. to avoid this error all fileTypes used by this filename must have the same eventDelimiter in their regex config.\n", filename, eventDelimiter)
				}
			}
			regexParserConfig.FieldExtractors = append(regexParserConfig.FieldExtractors, t.Regex.FieldExtractors...)
		} else {
			log.Printf("unhandled parser type=%v for filename=%v\n", t.ParserType, filename)
		}
	}

	if readInterval == nil {
		readInterval = &defaultReadInterval
	}

	return &IndexedFileConfig{
		Filename:     filename,
		ReadInterval: *readInterval,
		TimeLayout:   timeLayout,
		FileParser: &parser.RegexFileParser{
			Cfg: regexParserConfig,
		},
	}, nil
}

func ReadDynamicFileConfig(dynamicConfig config.DynamicConfig) ([]IndexedFileConfig, error) {
	fileTypeCfg, err := config.GetFileTypeConfig(dynamicConfig)
	if err != nil {
		return nil, fmt.Errorf("got error when getting file type config: %w", err)
	}
	filesCfg, _ := dynamicConfig.GetArray("files", []interface{}{}).Get()
	indexedFiles := make([]IndexedFileConfig, 0, len(filesCfg))
	for i, file := range filesCfg {
		fileMap, ok := file.(map[string]interface{})
		if !ok {
			log.Printf("failed to convert file at index=%v to map[string]interface{}. this file will be skipped. file=%v\n", i, file)
			continue
		}
		fn, ok := fileMap["fileName"]
		if !ok {
			log.Printf("did not get a fileName for file at index=%v\n", i)
		}
		filename, ok := fn.(string)
		if !ok {
			log.Printf("failed to convert filename to string for file at index=%v. fn=%v\n", i, fn)
		}
		typesCfg, ok := fileMap["fileTypes"]
		types := []string{"DEFAULT"}
		if !ok {
			log.Printf("did not get any fileTypes for filename=%v. will only use default config for these files.\n", fn)
		} else {
			typesCfgArr, ok := typesCfg.([]interface{})
			if !ok {
				log.Printf("failed to convert fileTypes to []interface{} for filename=%v. will only use default config for these files.\n", fn)
			} else {
				for j, ti := range typesCfgArr {
					ts, ok := ti.(string)
					if !ok {
						log.Printf("failed to convert fileType at index=%v to string for filename=%v. will not use this fileType for these files.\n", j, fn)
					} else {
						types = append(types, ts)
					}
				}
			}
		}
		fileTypes := make([]config.FileTypeConfig, 0, len(types))
		for _, tn := range types {
			t, ok := fileTypeCfg[tn]
			if !ok {
				log.Printf("did not find filetype=%v for filename=%v. will ignore this filetype.", tn, fn)
				continue
			}
			fileTypes = append(fileTypes, t)
		}
		ifc, err := mergeConfigs(filename, fileTypes)
		if err != nil {
			log.Printf("failed to merge configs for filename=%v. this file will be skipped: %v\n", filename, err)
			continue
		}
		indexedFiles = append(indexedFiles, *ifc)
	}
	return indexedFiles, nil
}
