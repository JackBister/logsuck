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

	fields := parser.ExtractFields(strings.ToLower(evt.Raw), ep.cfg.FieldExtractors)
	if t, ok := fields["_time"]; ok {
		parsed, err := time.Parse(ep.cfg.TimeLayout, t)
		if err != nil {
			processed.Timestamp = time.Now()
		} else {
			processed.Timestamp = parsed
		}
	} else {
		processed.Timestamp = time.Now()
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
