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
	"encoding/json"
	"fmt"
	"regexp"

	"go.uber.org/zap"
)

type JsonParserConfig struct {
	EventDelimiter *regexp.Regexp

	TimeField string
}

type JsonFileParser struct {
	Cfg JsonParserConfig

	Logger *zap.Logger
}

func (p *JsonFileParser) CanSplit(b []byte) bool {
	return p.Cfg.EventDelimiter.Match(b)
}

func (p *JsonFileParser) Extract(s string) (*ExtractResult, error) {
	fields := map[string]any{}
	err := json.Unmarshal([]byte(s), &fields)
	if err != nil {
		return nil, fmt.Errorf("error extracting fields from JSON string: %w", err)
	}
	fieldsConverted := map[string]string{}
	for k, v := range fields {
		if f, ok := v.(float64); ok {
			fieldsConverted[k] = fmt.Sprintf("%f", f)
		} else if f, ok := v.(float32); ok {
			fieldsConverted[k] = fmt.Sprintf("%f", f)
		} else {
			fieldsConverted[k] = fmt.Sprint(v)
		}
	}
	if _, ok := fieldsConverted[p.Cfg.TimeField]; ok {
		fieldsConverted["_time"] = fieldsConverted[p.Cfg.TimeField]
	}
	return &ExtractResult{
		Fields: fieldsConverted,
	}, nil
}

func (p *JsonFileParser) Split(s string) SplitResult {
	delimiters := p.Cfg.EventDelimiter.FindAllString(s, -1)
	split := p.Cfg.EventDelimiter.Split(s, -1)
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
