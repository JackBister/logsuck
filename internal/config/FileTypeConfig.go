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
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/parser"
	"go.uber.org/zap"
)

type ParserType = int

const (
	ParserTypeRegex ParserType = 1
)

type FileTypeConfig struct {
	Name         string
	TimeLayout   string
	ReadInterval time.Duration
	ParserType   ParserType

	Regex *parser.RegexParserConfig
}

const defaultEventDelimiter = "\n"

var defaultEventDelimiterRegexp = regexp.MustCompile(defaultEventDelimiter)
var defaultFieldExtractors = []*regexp.Regexp{
	regexp.MustCompile("(\\w+)=(\\w+)"),
	regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)"),
}

var defaultRegexParserConfig = parser.RegexParserConfig{
	EventDelimiter:  defaultEventDelimiterRegexp,
	FieldExtractors: defaultFieldExtractors,
}

const defaultTimeLayout = "2006/01/02 15:04:05"

func FileTypeConfigFromJSON(jsonFileTypes []jsonFileTypeConfig, logger *zap.Logger) (map[string]FileTypeConfig, error) {
	var err error
	fileTypes := make(map[string]FileTypeConfig, len(jsonFileTypes))
	for _, ft := range jsonFileTypes {
		var readInterval time.Duration
		if ft.ReadInterval != "" {
			readInterval, err = time.ParseDuration(ft.ReadInterval)
			if err != nil {
				// TODO:
				logger.Error("failed to read config for fileType: failed to parse readInterval",
					zap.String("fileType", ft.Name),
					zap.String("readInterval", ft.ReadInterval))
				continue
			}
		} else {
			logger.Info("will use default readInterval for fileType",
				zap.String("fileType", ft.Name))
			readInterval = defaultReadInterval
		}

		var parserType ParserType
		var regexParserConfig *parser.RegexParserConfig
		if ft.Parser == nil {
			logger.Info("will use default parser config for fileType",
				zap.String("fileType", ft.Name))
			parserType = ParserTypeRegex
			regexParserConfig = &defaultRegexParserConfig
		} else {

			if ft.Parser.Type != "" && ft.Parser.Type != "Regex" {
				// TODO:
				logger.Error("failed to read config for fileType: parser.type was not 'Regex'",
					zap.String("fileType", ft.Name))
				continue
			}

			if ft.Parser.RegexConfig == nil {
				logger.Error("failed to read config for fileType: parser.regexConfig was nil",
					zap.String("fileType", ft.Name))
				continue
			}

			parserType = ParserTypeRegex
			eventDelimiter, err := regexp.Compile(ft.Parser.RegexConfig.EventDelimiter)
			if err != nil {
				logger.Error("failed to read config for fileType: failed to compile eventDelimiter regexp",
					zap.String("fileType", ft.Name),
					zap.Error(err))
			}

			fe := make([]*regexp.Regexp, 0, len(ft.Parser.RegexConfig.FieldExtractors))
			for i, s := range ft.Parser.RegexConfig.FieldExtractors {
				rex, err := regexp.Compile(s)
				if err != nil {
					logger.Error("failed to read config for fileType: failed to compile fieldExtractor regexp",
						zap.String("fileType", ft.Name),
						zap.Int("index", i),
						zap.Error(err))
					continue
				}
				fe = append(fe, rex)
			}

			regexParserConfig = &parser.RegexParserConfig{
				EventDelimiter: eventDelimiter,

				FieldExtractors: fe,
			}
		}

		fileTypes[ft.Name] = FileTypeConfig{
			Name:         ft.Name,
			TimeLayout:   ft.TimeLayout,
			ReadInterval: readInterval,
			ParserType:   parserType,
			Regex:        regexParserConfig,
		}
	}
	if _, ok := fileTypes["DEFAULT"]; !ok {
		fileTypes["DEFAULT"] = FileTypeConfig{
			Name:         "DEFAULT",
			TimeLayout:   defaultTimeLayout,
			ReadInterval: defaultReadInterval,
			ParserType:   ParserTypeRegex,
			Regex:        &defaultRegexParserConfig,
		}
	}
	return fileTypes, nil
}
