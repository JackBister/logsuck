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
	"strconv"
	"strings"

	"github.com/jackbister/logsuck/pkg/logsuck/indexedfiles"
	"github.com/jackbister/logsuck/pkg/logsuck/parser"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type SurroundingPipelineStep struct {
	eventId int64
	count   int
}

func (s *SurroundingPipelineStep) Execute(ctx context.Context, pipe pipeline.Pipe, params pipeline.Parameters) {
	defer close(pipe.Output)

	evts, err := params.EventsRepo.GetSurroundingEvents(s.eventId, s.count)
	if err != nil {
		params.Logger.Error("got error when executing surrounding pipeline step",
			slog.Any("error", err)) // TODO: This needs to make it to the frontend somehow
		return
	}

	cfg, err := params.ConfigSource.Get()
	if err != nil {
		params.Logger.Error("got error when executing surrounding pipeline step: failed to get config",
			slog.Any("error", err))
		return
	}
	indexedFileConfigs, err := indexedfiles.ReadFileConfig(&cfg.Cfg, params.Logger)
	if err != nil {
		// TODO: signal error to rest of pipe??
		return
	}
	sourceToIfc := getSourceToIndexedFileConfig(evts, indexedFileConfigs)
	retEvts := make([]events.EventWithExtractedFields, len(evts))
	for i, evt := range evts {
		ifc, ok := sourceToIfc[evt.Source]
		if !ok {
			// TODO: How does the user get feedback about this?
			params.Logger.Warn("failed to find file configuration for event, this event will be ignored",
				slog.String("source", evt.Source))
			continue
		}
		evtFields, _ := parser.ExtractFields(strings.ToLower(evt.Raw), ifc.FileParser)
		retEvts[i] = events.EventWithExtractedFields{
			Id:        evt.Id,
			Raw:       evt.Raw,
			Timestamp: evt.Timestamp,
			Host:      evt.Host,
			Source:    evt.Source,
			SourceId:  evt.SourceId,
			Fields:    evtFields,
		}
	}
	pipe.Output <- pipeline.StepResult{
		Events: retEvts,
	}
}

func (s *SurroundingPipelineStep) Name() string {
	return "surrounding"
}

func (r *SurroundingPipelineStep) InputType() pipeline.PipeType {
	return pipeline.PipeTypeNone
}

func (r *SurroundingPipelineStep) OutputType() pipeline.PipeType {
	return pipeline.PipeTypeEvents
}

func (r *SurroundingPipelineStep) SortMode() events.SortMode {
	return events.SortModePreserveArgOrder
}

func compileSurroundingStep(input string, options map[string]string) (pipeline.Step, error) {
	eventIdString, ok := options["eventId"]
	if !ok {
		return nil, fmt.Errorf("failed to compile surrounding: eventId must be provided")
	}
	eventId, err := strconv.ParseInt(eventIdString, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to compile surrounding: failed to parse eventId as integer: %w", err)
	}
	var count int
	countString, ok := options["count"]
	if !ok {
		count = 100
	} else {
		count, err = strconv.Atoi(countString)
		if err != nil {
			return nil, fmt.Errorf("failed to compile surrounding: failed to parse count as integer: %w", err)
		}
	}
	return &SurroundingPipelineStep{
		eventId: eventId,
		count:   count,
	}, nil
}
