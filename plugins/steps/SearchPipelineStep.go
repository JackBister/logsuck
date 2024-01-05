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

package steps

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/araddon/dateparse"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/parser"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
	"github.com/jackbister/logsuck/pkg/logsuck/search"
)

type SearchPipelineStep struct {
	Search             *search.Search
	StartTime, EndTime *time.Time
}

func (s *SearchPipelineStep) Execute(ctx context.Context, pipe pipeline.Pipe, params pipeline.Parameters) {
	defer close(pipe.Output)

	cfg, err := params.ConfigSource.Get()
	if err != nil {
		params.Logger.Error("got error when executing search pipeline step: failed to get config",
			slog.Any("error", err))
		return
	}

	inputEvents := params.EventsRepo.FilterStream(s.Search, s.StartTime, s.EndTime)
	compiledFrags := compileKeys(s.Search.Fragments, params.Logger)
	compiledNotFrags := compileKeys(s.Search.NotFragments, params.Logger)
	compiledFields := compileFieldValues(s.Search.Fields, params.Logger)
	compiledNotFields := compileFieldValues(s.Search.NotFields, params.Logger)

	for {
		select {
		case <-ctx.Done():
			return
		case evts, ok := <-inputEvents:
			if !ok {
				return
			}
			indexedFiles, err := indexedfiles.ReadFileConfig(&cfg.Cfg, params.Logger)
			if err != nil {
				// TODO: signal error to rest of pipe??
				return
			}
			sourceToIfc := getSourceToIndexedFileConfig(evts, indexedFiles)
			retEvts := make([]events.EventWithExtractedFields, 0)
			for _, evt := range evts {
				ifc, ok := sourceToIfc[evt.Source]
				if !ok {
					// TODO: How does the user get feedback about this?
					params.Logger.Warn("failed to find file configuration for event, this event will be ignored",
						slog.String("source", evt.Source))
					continue
				}
				evtFields, include := shouldIncludeEvent(evt, ifc.FileParser, compiledFrags, compiledNotFrags, compiledFields, compiledNotFields)
				if include {
					retEvts = append(retEvts, events.EventWithExtractedFields{
						Id:        evt.Id,
						Raw:       evt.Raw,
						Timestamp: evt.Timestamp,
						Host:      evt.Host,
						Source:    evt.Source,
						SourceId:  evt.SourceId,
						Fields:    evtFields,
					})
				}
			}
			pipe.Output <- pipeline.StepResult{
				Events: retEvts,
			}
		}
	}
}

func (s *SearchPipelineStep) Name() string {
	return "search"
}

func (r *SearchPipelineStep) InputType() pipeline.PipeType {
	return pipeline.PipeTypeNone
}

func (r *SearchPipelineStep) OutputType() pipeline.PipeType {
	return pipeline.PipeTypeEvents
}

func compileSearchStep(input string, options map[string]string) (pipeline.Step, error) {
	var startTime, endTime *time.Time
	if t, ok := options["startTime"]; ok {
		startTimeParsed, err := dateparse.ParseStrict(t)
		if err != nil {
			return nil, fmt.Errorf("failed to create search: error parsing startTime: %w", err)
		}
		startTime = &startTimeParsed
	}
	if t, ok := options["endTime"]; ok {
		endTimeParsed, err := dateparse.ParseStrict(t)
		if err != nil {
			return nil, fmt.Errorf("failed to create search: error parsing endTime: %w", err)
		}
		endTime = &endTimeParsed
	}

	srch, err := parser.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create search: %w", err)
	}
	return &SearchPipelineStep{
		Search:    srch,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}
