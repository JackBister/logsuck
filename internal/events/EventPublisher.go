package events

import (
	"github.com/jackbister/logsuck/internal/parser"
	"log"
	"strings"
)

type EventPublisher interface {
	PublishEvent(evt RawEvent)
}

type repositoryPublisher struct {
	repository Repository
}

func RepositoryEventPublisher(repository Repository) EventPublisher {
	return &repositoryPublisher{
		repository: repository,
	}
}

func (ep *repositoryPublisher) PublishEvent(evt RawEvent) {
	processed := Event{
		Raw:       evt.Raw,
		Timestamp: evt.Timestamp,
		Source:    evt.Source,
	}

	parseResult, err := parser.Parse(strings.ToLower(evt.Raw), parser.ParseModeIngest)
	if err != nil {
		log.Println("Parsing failed with error=" + err.Error() + ", event will not be enriched. Raw=" + evt.Raw)
	} else if parseResult == nil {
		log.Println("Unexpected state: err != nil && parseResult == nil. event will not be enriched. Raw=" + evt.Raw)
	} else {
		processed.Fields = parseResult.Fields
	}

	ep.repository.Add(processed)
}

type debugEventPublisher struct {
	wrapped EventPublisher
}

func DebugEventPublisher(wrapped EventPublisher) EventPublisher {
	return &debugEventPublisher{
		wrapped: wrapped,
	}
}

func (ep *debugEventPublisher) PublishEvent(evt RawEvent) {
	log.Println("Received event:", evt)
	if ep.wrapped != nil {
		ep.wrapped.PublishEvent(evt)
	}
}

type nopEventPublisher struct {
}

func NopEventPublisher() EventPublisher {
	return &nopEventPublisher{}
}

func (ep *nopEventPublisher) PublishEvent(_ RawEvent) {}
