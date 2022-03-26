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
	"log"
	"strconv"
	"strings"

	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/internal/parser"
)

type surroundingPipelineStep struct {
	eventId int64
	count   int
}

func (s *surroundingPipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)

	evts, err := params.EventsRepo.GetSurroundingEvents(s.eventId, s.count)
	if err != nil {
		log.Printf("got error when executing surrounding pipeline step: %v", err) // TODO: This needs to make it to the frontend somehow
		return
	}

	indexedFileConfigs, err := indexedfiles.ReadDynamicFileConfig(params.DynamicConfig)
	if err != nil {
		// TODO: signal error to rest of pipe??
		return
	}
	sourceToIfc := getSourceToIndexedFileConfig(evts, indexedFileConfigs)
	retEvts := make([]events.EventWithExtractedFields, len(evts))
	for i, evt := range evts {
		ifc, ok := sourceToIfc[evt.Source]
		if !ok {
			// TODO: Error or automatically get default IFC?
		}
		evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), ifc.FileParser)
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
	pipe.output <- PipelineStepResult{
		Events: retEvts,
	}
}

func (s *surroundingPipelineStep) IsGeneratorStep() bool {
	return true
}

func (s *surroundingPipelineStep) Name() string {
	return "surrounding"
}

func compileSurroundingStep(input string, options map[string]string) (pipelineStep, error) {
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
	return &surroundingPipelineStep{
		eventId: eventId,
		count:   count,
	}, nil
}
