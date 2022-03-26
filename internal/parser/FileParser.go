package parser

import (
	"log"
	"regexp"
)

type RegexParserConfig struct {
	EventDelimiter  *regexp.Regexp
	FieldExtractors []*regexp.Regexp
}

type RawParserEvent struct {
	Raw    string
	Offset int64
}

type ExtractResult struct {
	Fields map[string]string
}

type SplitResult struct {
	Events    []RawParserEvent
	Remainder string
}

type FileParser interface {
	CanSplit(b []byte) bool
	Extract(s string) ExtractResult
	Split(s string) SplitResult
}

type RegexFileParser struct {
	Cfg RegexParserConfig
}

func (r *RegexFileParser) CanSplit(b []byte) bool {
	return r.Cfg.EventDelimiter.Match(b)
}

func (r *RegexFileParser) Extract(s string) ExtractResult {
	ret := map[string]string{}
	for _, rex := range r.Cfg.FieldExtractors {
		subExpNames := rex.SubexpNames()[1:]
		isNamedOnlyExtractor := true
		for _, name := range subExpNames {
			if name == "" {
				isNamedOnlyExtractor = false
			}
		}
		matches := rex.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			if isNamedOnlyExtractor && len(rex.SubexpNames()) == len(match) {
				for j, name := range subExpNames {
					ret[name] = match[j+1]
				}
			} else if len(match) == 3 {
				ret[match[1]] = match[2]
			} else {
				log.Printf("Malformed field extractor '%v': If there are any unnamed capture groups in the regex, there must be exactly two capture groups.\n", rex)
			}
		}
	}
	return ExtractResult{
		Fields: ret,
	}
}

func (r *RegexFileParser) Split(s string) SplitResult {
	delimiters := r.Cfg.EventDelimiter.FindAllString(s, -1)
	split := r.Cfg.EventDelimiter.Split(s, -1)
	rawEvts := split[:len(split)-1]
	retEvts := make([]RawParserEvent, 0, len(rawEvts))
	offset := int64(0)
	for i, raw := range rawEvts {
		evt := RawParserEvent{
			Raw:    raw,
			Offset: int64(offset),
		}
		retEvts = append(retEvts, evt)
		offset += int64(len(raw)) + int64(len(delimiters[i]))
	}
	return SplitResult{
		Events:    retEvts,
		Remainder: split[len(split)-1],
	}
}