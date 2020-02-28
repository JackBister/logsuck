package events

import (
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
	"log"
	"strings"
	"time"
)

type EventPublisher interface {
	PublishEvent(evt RawEvent)
}

type repositoryPublisher struct {
	cfg        *config.Config
	repository Repository
}

func RepositoryEventPublisher(cfg *config.Config, repository Repository) EventPublisher {
	return &repositoryPublisher{
		cfg:        cfg,
		repository: repository,
	}
}

func (ep *repositoryPublisher) PublishEvent(evt RawEvent) {
	processed := Event{
		Raw:    evt.Raw,
		Source: evt.Source,
	}

	parseResult, err := parser.Parse(strings.ToLower(evt.Raw), parser.ParseModeIngest, ep.cfg)
	if err != nil {
		processed.Fields = ep.createDefaultFields(evt)
		processed.Timestamp = time.Now()
		log.Println("Parsing failed with error=" + err.Error() + ", event will not be enriched. Raw=" + evt.Raw)
	} else if parseResult == nil {
		processed.Fields = ep.createDefaultFields(evt)
		processed.Timestamp = time.Now()
		log.Println("Unexpected state: err != nil && parseResult == nil. event will not be enriched. Raw=" + evt.Raw)
	} else {
		processed.Fields = parseResult.Fields
		processed.Fields["source"] = evt.Source
		processed.Timestamp = parseResult.Time
	}

	ep.repository.Add(processed)
}

func (ep *repositoryPublisher) createDefaultFields(evt RawEvent) map[string]string {
	return map[string]string{
		"_time":  time.Now().Format(ep.cfg.TimeLayout),
		"source": evt.Source,
	}
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
