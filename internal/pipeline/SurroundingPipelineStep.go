package pipeline

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jackbister/logsuck/internal/events"
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

	retEvts := make([]events.EventWithExtractedFields, len(evts))
	for i, evt := range evts {
		evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), params.Cfg.FieldExtractors)
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