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
		timeLayout, _ := fileTypeCfg.GetString("timeLayout", defaultTimeLayout).Get()
		readIntervalString, _ := fileTypeCfg.GetString("readInterval", defaultReadInterval).Get()
		readInterval, err := time.ParseDuration(readIntervalString)
		if err != nil {
			log.Printf("failed to parse duration=%v for file type with key=%v. will use defaultReadInterval=%v\n", readIntervalString, k, defaultReadInterval)
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
				log.Printf("failed to convert regex parser config for file type with key=%v. this file type will not be usable: %v\n", k, err)
				continue
			}
			regexParserConfig = r
		} else {
			log.Printf("got unknown parser type=%v for file type with key=%v. this file type will not be usable.\n", parserTypeString, k)
			continue
		}
		ret[k] = FileTypeConfig{
			TimeLayout:   timeLayout,
			ReadInterval: readInterval,
			ParserType:   parserType,
			Regex:        regexParserConfig,
		}
	}
	return ret, nil
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
