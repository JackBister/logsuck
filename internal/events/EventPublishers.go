package events

import (
	"log/slog"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/parser"
	"github.com/jackbister/logsuck/pkg/logsuck/config"
	api "github.com/jackbister/logsuck/pkg/logsuck/events"
	"go.uber.org/dig"
)

type batchedRepositoryPublisher struct {
	cfg  *config.Config
	repo api.Repository

	adder chan api.Event

	logger *slog.Logger
}

type BatchedRepositoryPublisherParams struct {
	dig.In

	Cfg    *config.Config
	Repo   api.Repository
	Logger *slog.Logger
}

func BatchedRepositoryPublisher(p BatchedRepositoryPublisherParams) api.Publisher {
	adder := make(chan api.Event, 5000)

	go func() {
		accumulated := make([]api.Event, 0, 5000)
		timeout := time.After(1 * time.Second)
		for {
			select {
			case <-timeout:
				if len(accumulated) > 0 {
					p.Repo.AddBatch(accumulated)
					accumulated = accumulated[:0]
				}
				timeout = time.After(1 * time.Second)
			case evt := <-adder:
				accumulated = append(accumulated, evt)
				if len(accumulated) >= 5000 {
					err := p.Repo.AddBatch(accumulated)
					if err != nil {
						// TODO: Error handling
						p.Logger.Error("error when adding events",
							slog.Any("error", err))
					}
					accumulated = accumulated[:0]
					timeout = time.After(1 * time.Second)
				}
			}
		}
	}()

	return &batchedRepositoryPublisher{
		cfg:  p.Cfg,
		repo: p.Repo,

		adder: adder,

		logger: p.Logger,
	}
}

func (ep *batchedRepositoryPublisher) PublishEvent(evt api.RawEvent, timeLayout string, fileParser parser.FileParser) {
	processed := api.Event{
		Raw:      evt.Raw,
		Host:     ep.cfg.HostName,
		SourceId: evt.SourceId,
		Source:   evt.Source,
		Offset:   evt.Offset,
	}

	fields, err := parser.ExtractFields(strings.ToLower(evt.Raw), fileParser)
	if err != nil {
		ep.logger.Warn("failed to extract fields when getting timestamp, will use current time as timestamp",
			slog.String("fileName", evt.Source),
			slog.Any("error", err))
		processed.Timestamp = time.Now()
	} else if t, ok := fields["_time"]; ok {
		parsed, err := parser.ParseTime(timeLayout, t)
		if err != nil {
			ep.logger.Warn("failed to parse _time field, will use current time as timestamp",
				slog.String("fileName", evt.Source),
				slog.Any("error", err))
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
	repository api.Repository
}

type nopEventPublisher struct {
}

func NopEventPublisher() api.Publisher {
	return &nopEventPublisher{}
}

func (ep *nopEventPublisher) PublishEvent(_ api.RawEvent, _ string, _ parser.FileParser) {}
