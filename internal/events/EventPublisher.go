package events

import (
	"log"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
)

type EventPublisher interface {
	PublishEvent(evt RawEvent)
}

type batchedRepositoryPublisher struct {
	cfg  *config.Config
	repo Repository

	adder chan<- Event
}

func BatchedRepositoryPublisher(cfg *config.Config, repo Repository) EventPublisher {
	adder := make(chan Event)

	go func() {
		accumulated := make([]Event, 0, 1000)
		timeout := time.After(1 * time.Second)
		for {
			select {
			case <-timeout:
				if len(accumulated) > 0 {
					repo.AddBatch(accumulated)
					accumulated = accumulated[:0]
				}
				timeout = time.After(1 * time.Second)
			case evt := <-adder:
				accumulated = append(accumulated, evt)
				if len(accumulated) >= 1000 {
					_, err := repo.AddBatch(accumulated)
					if err != nil {
						// TODO: Error handling
						log.Println("error when adding events:", err)
					}
					accumulated = accumulated[:0]
					timeout = time.After(1 * time.Second)
				}
			}
		}
	}()

	return &batchedRepositoryPublisher{
		cfg:  cfg,
		repo: repo,

		adder: adder,
	}
}

func (ep *batchedRepositoryPublisher) PublishEvent(evt RawEvent) {
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

	ep.adder <- processed
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
