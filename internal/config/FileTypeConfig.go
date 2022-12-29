package config

import (
	"log"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/parser"
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

func FileTypeConfigFromJSON(jsonFileTypes map[string]jsonFileTypeConfig) (map[string]FileTypeConfig, error) {
	var err error
	fileTypes := make(map[string]FileTypeConfig, len(jsonFileTypes))
	for name, ft := range jsonFileTypes {
		var readInterval time.Duration
		if ft.ReadInterval != "" {
			readInterval, err = time.ParseDuration(ft.ReadInterval)
			if err != nil {
				// TODO:
				log.Printf("failed to read config for filetype with name=%v: failed to parse ReadInterval=%v\n", name, ft.ReadInterval)
				continue
			}
		} else {
			log.Printf("will use default readInterval for filetype with name=%v", name)
			readInterval = defaultReadInterval
		}

		var parserType ParserType
		var regexParserConfig *parser.RegexParserConfig
		if ft.Parser == nil {
			log.Printf("will use default paser config for filetype with name=%v", name)
			parserType = ParserTypeRegex
			regexParserConfig = &defaultRegexParserConfig
		} else {

			if ft.Parser.Type != "" && ft.Parser.Type != "Regex" {
				// TODO:
				log.Printf("failed to read config for filetype with name=%v: parser.type was not 'Regex'\n", name)
				continue
			}

			if ft.Parser.RegexConfig == nil {
				log.Printf("failed to read config for filetype with name=%v: parser.regexConfig was nil\n", name)
				continue
			}

			parserType = ParserTypeRegex
			eventDelimiter, err := regexp.Compile(ft.Parser.RegexConfig.EventDelimiter)
			if err != nil {
				log.Printf("failed to read config for filetype with name=%v: failed to compile eventDelimiter regexp: %v\n", name, err)
			}

			fe := make([]*regexp.Regexp, 0, len(ft.Parser.RegexConfig.FieldExtractors))
			for i, s := range ft.Parser.RegexConfig.FieldExtractors {
				rex, err := regexp.Compile(s)
				if err != nil {
					log.Printf("failed to read config for filetype with name=%v: failed to compile fieldExtractor regexp at index=%v: %v\n", name, i, err)
					continue
				}
				fe = append(fe, rex)
			}

			regexParserConfig = &parser.RegexParserConfig{
				EventDelimiter: eventDelimiter,

				FieldExtractors: fe,
			}
		}

		fileTypes[name] = FileTypeConfig{
			Name:         name,
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
