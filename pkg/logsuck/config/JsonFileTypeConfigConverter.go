package config

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

const defaultEventDelimiter = "\n"
const defaultReadInterval = 1 * time.Second
const defaultTimeField = "_time"

var defaultEventDelimiterRegexp = regexp.MustCompile(defaultEventDelimiter)
var defaultFieldExtractors = []*regexp.Regexp{
	regexp.MustCompile("(\\w+)=(\\w+)"),
	regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)"),
}

var defaultRegexParserConfig = RegexParserConfig{
	EventDelimiter:  defaultEventDelimiterRegexp,
	FieldExtractors: defaultFieldExtractors,
	TimeField:       defaultTimeField,
}

const defaultTimeLayout = "2006/01/02 15:04:05"

func FileTypeConfigFromJSON(jsonFileTypes []jsonFileTypeConfig, logger *slog.Logger) (map[string]FileTypeConfig, error) {
	var err error
	fileTypes := make(map[string]FileTypeConfig, len(jsonFileTypes))
	for _, ft := range jsonFileTypes {
		var readInterval time.Duration
		if ft.ReadInterval != "" {
			readInterval, err = time.ParseDuration(ft.ReadInterval)
			if err != nil {
				logger.Error("failed to read config for fileType: failed to parse readInterval",
					slog.String("fileType", ft.Name),
					slog.String("readInterval", ft.ReadInterval))
				return nil, fmt.Errorf("failed to read config for fileType: failed to parse readInterval")
			}
		} else {
			logger.Info("will use default readInterval for fileType",
				slog.String("fileType", ft.Name))
			readInterval = defaultReadInterval
		}

		var parserType ParserType
		var jsonParserConfig *JsonParserConfig
		var regexParserConfig *RegexParserConfig
		if ft.Parser == nil {
			logger.Info("will use default parser config for fileType",
				slog.String("fileType", ft.Name))
			parserType = ParserTypeRegex
			regexParserConfig = &defaultRegexParserConfig
		} else if ft.Parser.Type == "JSON" {
			if ft.Parser.JsonConfig == nil {
				logger.Error("failed to read config for fileType: parser.jsonConfig was nil",
					slog.String("fileType", ft.Name))
				return nil, fmt.Errorf("failed to read config for fileType: parser.jsonConfig was nil")
			}

			parserType = ParserTypeJSON
			eventDelimiter, err := regexp.Compile(ft.Parser.JsonConfig.EventDelimiter)
			if err != nil {
				logger.Error("failed to read config for fileType: failed to compile eventDelimiter regexp",
					slog.String("fileType", ft.Name),
					slog.Any("error", err))
			}

			jsonParserConfig = &JsonParserConfig{
				EventDelimiter: eventDelimiter,
				TimeField:      ft.Parser.JsonConfig.TimeField,
			}
		} else if ft.Parser.Type == "Regex" {
			if ft.Parser.RegexConfig == nil {
				logger.Error("failed to read config for fileType: parser.regexConfig was nil",
					slog.String("fileType", ft.Name))
				return nil, fmt.Errorf("failed to read config for fileType: parser.regexConfig was nil")
			}

			parserType = ParserTypeRegex
			eventDelimiter, err := regexp.Compile(ft.Parser.RegexConfig.EventDelimiter)
			if err != nil {
				logger.Error("failed to read config for fileType: failed to compile eventDelimiter regexp",
					slog.String("fileType", ft.Name),
					slog.Any("error", err))
				return nil, fmt.Errorf("failed to read config for fileType: failed to compile eventDelimiter regexp")
			}

			fe := make([]*regexp.Regexp, 0, len(ft.Parser.RegexConfig.FieldExtractors))
			for i, s := range ft.Parser.RegexConfig.FieldExtractors {
				rex, err := regexp.Compile(s)
				if err != nil {
					logger.Error("failed to read config for fileType: failed to compile fieldExtractor regexp",
						slog.String("fileType", ft.Name),
						slog.Int("index", i),
						slog.Any("error", err))
					continue
				}
				fe = append(fe, rex)
			}

			timeField := defaultTimeField
			if ft.Parser.RegexConfig.TimeField != "" {
				timeField = ft.Parser.RegexConfig.TimeField
			} else {
				logger.Warn("got empty timeField for fileType, will use default timeField",
					slog.String("fileType", ft.Name),
					slog.String("defaultTimeField", defaultTimeField))
			}

			regexParserConfig = &RegexParserConfig{
				EventDelimiter:  eventDelimiter,
				FieldExtractors: fe,
				TimeField:       timeField,
			}
		} else {
			logger.Error("failed to read config for fileType: Unknown parser.type",
				slog.String("fileType", ft.Name),
				slog.String("parserType", ft.Parser.Type))
			return nil, fmt.Errorf("failed to read config for fileType: Unknown parser.type")
		}

		fileTypes[ft.Name] = FileTypeConfig{
			Name:         ft.Name,
			TimeLayout:   ft.TimeLayout,
			ReadInterval: readInterval,
			ParserType:   parserType,
			JSON:         jsonParserConfig,
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
