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

package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/araddon/dateparse"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/parser"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	api "github.com/jackbister/logsuck/pkg/logsuck/pipeline"
	"github.com/jackbister/logsuck/pkg/logsuck/search"
)

type searchPipelineStep struct {
	srch               *search.Search
	startTime, endTime *time.Time
}

func (s *searchPipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)

	cfg, err := params.ConfigSource.Get()
	if err != nil {
		params.Logger.Error("got error when executing search pipeline step: failed to get config",
			slog.Any("error", err))
		return
	}

	inputEvents := params.EventsRepo.FilterStream(s.srch, s.startTime, s.endTime)
	compiledFrags := compileKeys(s.srch.Fragments, params.Logger)
	compiledNotFrags := compileKeys(s.srch.NotFragments, params.Logger)
	compiledFields := compileFieldValues(s.srch.Fields, params.Logger)
	compiledNotFields := compileFieldValues(s.srch.NotFields, params.Logger)

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
			pipe.output <- PipelineStepResult{
				Events: retEvts,
			}
		}
	}
}

func (s *searchPipelineStep) Name() string {
	return "search"
}

func (r *searchPipelineStep) InputType() api.PipelinePipeType {
	return api.PipelinePipeTypeNone
}

func (r *searchPipelineStep) OutputType() api.PipelinePipeType {
	return api.PipelinePipeTypeEvents
}

func compileSearchStep(input string, options map[string]string) (pipelineStep, error) {
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
	return &searchPipelineStep{
		srch:      srch,
		startTime: startTime,
		endTime:   endTime,
	}, nil
}
