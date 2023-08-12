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

package parser

import (
	"log/slog"
	"regexp"
)

type RegexParserConfig struct {
	EventDelimiter  *regexp.Regexp
	FieldExtractors []*regexp.Regexp
	TimeField       string
}

type RegexFileParser struct {
	Cfg RegexParserConfig

	Logger *slog.Logger
}

func (r *RegexFileParser) CanSplit(b []byte) bool {
	return r.Cfg.EventDelimiter.Match(b)
}

func (r *RegexFileParser) Extract(s string) (*ExtractResult, error) {
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
				r.Logger.Warn("Malformed field extractor': If there are any unnamed capture groups in the regex, there must be exactly two capture groups",
					slog.Any("fieldExtractor", rex))
			}
		}
	}
	if _, ok := ret[r.Cfg.TimeField]; ok {
		ret["_time"] = ret[r.Cfg.TimeField]
	}
	return &ExtractResult{
		Fields: ret,
	}, nil
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
