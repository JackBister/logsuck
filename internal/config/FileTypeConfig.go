package config

import (
	"fmt"
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
	TimeLayout   string
	ReadInterval time.Duration
	ParserType   ParserType

	Regex *parser.RegexParserConfig
}

const defaultEventDelimiter = "\n"

var defaultEventDelimiterRegexp = regexp.MustCompile(defaultEventDelimiter)
var defaultFieldExtractors = []*regexp.Regexp{
	regexp.MustCompile("(\\w+)=(\\w+)"),
	regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)"),
}

const defaultTimeLayout = "2006/01/02 15:04:05"

func GetFileTypeConfig(dynamicConfig DynamicConfig) (map[string]FileTypeConfig, error) {
	fileTypesCfg := dynamicConfig.Cd("fileTypes")
	keys, ok := fileTypesCfg.Ls(false)
	if !ok {
		return map[string]FileTypeConfig{}, fmt.Errorf("did not get ok when getting list of keys from DynamicConfig")
	}
	ret := make(map[string]FileTypeConfig, len(keys))
	for _, k := range keys {
		fileTypeCfg := fileTypesCfg.Cd(k)
		cfg, err := FileTypeConfigFromDynamicConfig(k, fileTypeCfg)
		if err != nil {
			log.Printf("got error when reading configuration for filetype=%v: %v\n", k, err)
			continue
		}
		ret[k] = *cfg
	}
	if _, ok := ret["DEFAULT"]; !ok {
		defaultReadIntervalDuration, err := time.ParseDuration(defaultReadInterval)
		if err != nil {
			panic("defaultReadInterval could not be parsed as a duration. this indicates that someone has seriously screwed up. you can probably work around this by adding a DEFAULT key to your fileTypes in the configuration.")
		}
		ret["DEFAULT"] = FileTypeConfig{
			TimeLayout:   defaultTimeLayout,
			ReadInterval: defaultReadIntervalDuration,
			ParserType:   ParserTypeRegex,
			Regex: &parser.RegexParserConfig{
				EventDelimiter:  defaultEventDelimiterRegexp,
				FieldExtractors: defaultFieldExtractors,
			},
		}
	}
	return ret, nil
}

func FileTypeConfigFromDynamicConfig(name string, fileTypeCfg DynamicConfig) (*FileTypeConfig, error) {
	timeLayout, _ := fileTypeCfg.GetString("timeLayout", defaultTimeLayout).Get()
	readIntervalString, _ := fileTypeCfg.GetString("readInterval", defaultReadInterval).Get()
	readInterval, err := time.ParseDuration(readIntervalString)
	if err != nil {
		log.Printf("failed to parse duration=%v for file type with key=%v. will use defaultReadInterval=%v\n", readIntervalString, name, defaultReadInterval)
		readInterval, _ = time.ParseDuration(defaultReadInterval)
	}
	parserCfg := fileTypeCfg.Cd("parser")
	parserTypeString, _ := parserCfg.GetString("type", "Regex").Get()
	var parserType ParserType
	var regexParserConfig *parser.RegexParserConfig
	if parserTypeString == "Regex" {
		parserType = ParserTypeRegex
		r, err := getRegexParserConfig(parserCfg.Cd("regexConfig"))
		if err != nil {
			return nil, fmt.Errorf("failed to convert regex parser config for file type with key=%v. this file type will not be usable: %v", name, err)
		}
		regexParserConfig = r
	} else {
		return nil, fmt.Errorf("got unknown parser type=%v for file type with key=%v. this file type will not be usable.\n", parserTypeString, name)
	}
	return &FileTypeConfig{
		TimeLayout:   timeLayout,
		ReadInterval: readInterval,
		ParserType:   parserType,
		Regex:        regexParserConfig,
	}, nil
}

func getRegexParserConfig(dynamicConfig DynamicConfig) (*parser.RegexParserConfig, error) {
	eventDelimiterString, _ := dynamicConfig.GetString("eventDelimiter", defaultEventDelimiter).Get()
	eventDelimiter, err := regexp.Compile(eventDelimiterString)
	if err != nil {
		return nil, fmt.Errorf("failed to compile event delimiter regex: %w", err)
	}

	fieldExtractorsArr, _ := dynamicConfig.GetArray("fieldExtractors", []interface{}{}).Get()
	fieldExtractors := make([]*regexp.Regexp, len(fieldExtractorsArr))
	for i, fieldExtractorInterface := range fieldExtractorsArr {
		fieldExtractorString, ok := fieldExtractorInterface.(string)
		if !ok {
			return nil, fmt.Errorf("failed to convert field extractor at index=%v to string. fieldExtractor=%v", i, fieldExtractorInterface)
		}
		fieldExtractor, err := regexp.Compile(fieldExtractorString)
		if err != nil {
			return nil, fmt.Errorf("failed to compile field extractor regex: %w", err)
		}
		fieldExtractors[i] = fieldExtractor
	}

	return &parser.RegexParserConfig{
		EventDelimiter:  eventDelimiter,
		FieldExtractors: fieldExtractors,
	}, nil
}
