// Copyright 2020 The Logsuck Authors
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

package filtering

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
	"github.com/jackbister/logsuck/internal/search"
)

func FilterEventsStream(ctx context.Context, repo events.Repository, srch *search.Search, cfg *config.Config) <-chan []events.EventWithExtractedFields {
	ret := make(chan []events.EventWithExtractedFields)

	go func() {
		defer close(ret)
		inputEvents := repo.FilterStream(srch)
		compiledFrags := CompileKeys(srch.Fragments)
		compiledNotFrags := CompileKeys(srch.NotFragments)
		compiledFields := CompileMap(srch.Fields)
		compiledNotFields := CompileMap(srch.NotFields)

		for evts := range inputEvents {
			retEvts := make([]events.EventWithExtractedFields, 0)
			for _, evt := range evts {
				evtFields, include := shouldIncludeEvent(evt, cfg, compiledFrags, compiledNotFrags, compiledFields, compiledNotFields)
				if include {
					retEvts = append(retEvts, events.EventWithExtractedFields{
						Id:        evt.Id,
						Raw:       evt.Raw,
						Timestamp: evt.Timestamp,
						Source:    evt.Source,
						Fields:    evtFields,
					})
				}
			}
			ret <- retEvts
		}
	}()
	return ret
}

func shouldIncludeEvent(evt events.EventWithId,
	cfg *config.Config,
	compiledFrags []*regexp.Regexp, compiledNotFrags []*regexp.Regexp,
	compiledFields map[string][]*regexp.Regexp, compiledNotFields map[string][]*regexp.Regexp) (map[string]string, bool) {
	rawLowered := strings.ToLower(evt.Raw)
	evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), cfg.FieldExtractors)
	// TODO: This could produce unexpected results
	evtFields["host"] = evt.Host
	evtFields["source"] = evt.Source

	include := true
	for _, frag := range compiledFrags {
		if !frag.MatchString(rawLowered) {
			include = false
			break
		}
	}
	for _, frag := range compiledNotFrags {
		if frag.MatchString(rawLowered) {
			include = false
			break
		}
	}
	for key, values := range compiledFields {
		evtValue, ok := evtFields[key]
		if !ok {
			include = false
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if !anyMatch {
			include = false
			break
		}
	}
	for key, values := range compiledNotFields {
		evtValue, ok := evtFields[key]
		if !ok {
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if anyMatch {
			include = false
			break
		}
	}
	return evtFields, include
}
