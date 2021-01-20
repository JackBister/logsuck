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

package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/araddon/dateparse"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/search"
)

type searchPipelineStep struct {
	srch               *search.Search
	startTime, endTime *time.Time
}

func (s *searchPipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)
	inputEvents := params.EventsRepo.FilterStream(s.srch, s.startTime, s.endTime)
	compiledFrags := compileKeys(s.srch.Fragments)
	compiledNotFrags := compileKeys(s.srch.NotFragments)
	compiledFields := compileFieldValues(s.srch.Fields)
	compiledNotFields := compileFieldValues(s.srch.NotFields)

	for {
		select {
		case <-ctx.Done():
			return
		case evts, ok := <-inputEvents:
			if !ok {
				return
			}
			retEvts := make([]events.EventWithExtractedFields, 0)
			for _, evt := range evts {
				evtFields, include := shouldIncludeEvent(evt, params.Cfg, compiledFrags, compiledNotFrags, compiledFields, compiledNotFields)
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
			pipe.output <- PipelineStepResult{
				Events: retEvts,
			}
		}
	}
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

	srch, err := search.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create search: %w", err)
	}
	return &searchPipelineStep{
		srch:      srch,
		startTime: startTime,
		endTime:   endTime,
	}, nil
}
