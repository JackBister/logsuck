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
	"log/slog"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/parser"
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

func mergeConfigs(filename string, fileTypes []config.FileTypeConfig, logger *slog.Logger) (*IndexedFileConfig, error) {
	var jsonParserConfig *config.JsonParserConfig
	var regexParserConfig *config.RegexParserConfig
	timeLayout := ""
	var readInterval *time.Duration
	for _, t := range fileTypes {
		if timeLayout == "" {
			timeLayout = t.TimeLayout
		} else {
			if t.Name != "DEFAULT" && timeLayout != t.TimeLayout {
				logger.Warn("encountered multiple timeLayouts for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same timeLayout.",
					slog.String("fileName", filename),
					slog.String("chosenTimeLayout", timeLayout),
					slog.String("discardedTimeLayout", t.TimeLayout))
			}
		}

		if readInterval == nil {
			readInterval = &t.ReadInterval
		} else {
			if t.Name != "DEFAULT" && readInterval != &t.ReadInterval {
				logger.Warn("encountered multiple readIntervals for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same readInterval.",
					slog.String("fileName", filename),
					slog.Duration("chosenReadInterval", *readInterval),
					slog.Duration("discardedReadInterval", t.ReadInterval))
			}
		}

		if t.JSON != nil {
			if jsonParserConfig == nil {
				jsonParserConfig = &config.JsonParserConfig{}
			}
			if regexParserConfig != nil {
				if t.Name == "DEFAULT" {
					continue
				}
				return nil, fmt.Errorf("Failed to merge fileType configs for file=%v: found conflicting parser types where one parser is JSON and another is Regex. Check your fileTypes for this file. fileTypes=%v",
					filename,
					fileTypes,
				)
			}
			eventDelimiter := t.JSON.EventDelimiter
			if jsonParserConfig.EventDelimiter == nil {
				jsonParserConfig.EventDelimiter = eventDelimiter
			} else if t.Name != "DEFAULT" && jsonParserConfig.EventDelimiter.String() != eventDelimiter.String() {
				logger.Warn("encountered multiple eventDelimiters for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same eventDelimiter.",
					slog.String("fileName", filename),
					slog.Any("chosenEventDelimiter", eventDelimiter),
					slog.Any("discardedEventDelimiter", jsonParserConfig.EventDelimiter))
				jsonParserConfig.EventDelimiter = eventDelimiter
			}

			timeField := t.JSON.TimeField
			if jsonParserConfig.TimeField == "" {
				jsonParserConfig.TimeField = timeField
			} else if t.Name != "DEFAULT" && jsonParserConfig.TimeField != timeField {
				logger.Warn("encountered multiple timeFields for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same timeField.",
					slog.String("fileName", filename),
					slog.String("chosenTimeField", timeField),
					slog.String("discardedTimeField", jsonParserConfig.TimeField))
				jsonParserConfig.TimeField = timeField
			}
		} else if t.Regex != nil {
			if regexParserConfig == nil {
				regexParserConfig = &config.RegexParserConfig{}
			}
			if jsonParserConfig != nil {
				if t.Name == "DEFAULT" {
					continue
				}
				return nil, fmt.Errorf("Failed to merge fileType configs for file=%v: found conflicting parser types where one parser is JSON and another is Regex. Check your fileTypes for this file. fileTypes=%v",
					filename,
					fileTypes,
				)
			}
			eventDelimiter := t.Regex.EventDelimiter
			if regexParserConfig.EventDelimiter == nil {
				regexParserConfig.EventDelimiter = eventDelimiter
			} else if t.Name != "DEFAULT" && regexParserConfig.EventDelimiter.String() != eventDelimiter.String() {
				logger.Warn("encountered multiple eventDelimiters for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same eventDelimiter.",
					slog.String("fileName", filename),
					slog.Any("chosenEventDelimiter", eventDelimiter),
					slog.Any("discardedEventDelimiter", regexParserConfig.EventDelimiter))
				regexParserConfig.EventDelimiter = eventDelimiter
			}

			regexParserConfig.FieldExtractors = append(regexParserConfig.FieldExtractors, t.Regex.FieldExtractors...)

			timeField := t.Regex.TimeField
			if regexParserConfig.TimeField == "" {
				regexParserConfig.TimeField = timeField
			} else if t.Name != "DEFAULT" && regexParserConfig.TimeField != timeField {
				logger.Warn("encountered multiple timeFields for file. will choose one of them. to avoid this error all fileTypes used by this filename must have the same timeField.",
					slog.String("fileName", filename),
					slog.String("chosenTimeField", timeField),
					slog.String("discardedTimeField", regexParserConfig.TimeField))
				regexParserConfig.TimeField = timeField
			}
		} else {
			logger.Error("unhandled parserType for file",
				slog.String("fileName", filename),
				slog.Int("parserType", t.ParserType))
		}
	}

	if timeLayout == "" {
		timeLayout = defaultTimeLayout
	}

	if readInterval == nil {
		readInterval = &defaultReadInterval
	}

	var fp parser.FileParser
	if jsonParserConfig != nil {
		fp = &parser.JsonFileParser{
			Cfg: *jsonParserConfig,

			Logger: logger,
		}
	} else if regexParserConfig != nil {
		fp = &parser.RegexFileParser{
			Cfg: *regexParserConfig,

			Logger: logger,
		}
	}

	return &IndexedFileConfig{
		Filename:     filename,
		ReadInterval: *readInterval,
		TimeLayout:   timeLayout,
		FileParser:   fp,
	}, nil
}

func ReadFileConfig(cfg *config.Config, logger *slog.Logger) ([]IndexedFileConfig, error) {
	fileTypes := cfg.FileTypes
	files := cfg.Files
	indexedFiles := make([]IndexedFileConfig, 0, len(fileTypes))
	hostType := cfg.HostTypes[cfg.HostType]
	defaultHostType := cfg.HostTypes["DEFAULT"]
	hostTypeFiles := append(hostType.Files, defaultHostType.Files...)
	for _, v := range hostTypeFiles {
		fileCfg, ok := files[v.Name]
		if !ok {
			logger.Error("Failed to find config for file. This file will be ignored.",
				slog.String("fileName", v.Name))
			continue
		}
		fileTypeCfgs := make([]config.FileTypeConfig, 0, len(fileCfg.Filetypes))
		for _, ftn := range append(fileCfg.Filetypes, "DEFAULT") {
			ftc, ok := fileTypes[ftn]
			if !ok {
				logger.Error("Failed to find fileType when configuring filen. This file will be indexed but may be incorrectly configured.",
					slog.String("fileType", ftn),
					slog.String("fileName", v.Name))
				continue
			}
			fileTypeCfgs = append(fileTypeCfgs, ftc)
		}
		ifc, err := mergeConfigs(v.Name, fileTypeCfgs, logger)
		if err != nil {
			logger.Error("Failed to merge configuration for file. This file will be ignored",
				slog.String("fileName", v.Name),
				slog.Any("error", err))
		}
		indexedFiles = append(indexedFiles, *ifc)
	}
	return indexedFiles, nil
}
