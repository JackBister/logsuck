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

package events

import (
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

type EventPublisher interface {
	PublishEvent(evt RawEvent, timeLayout string, fileParser parser.FileParser)
}

type batchedRepositoryPublisher struct {
	cfg  *config.Config
	repo Repository

	adder chan Event

	logger *zap.Logger
}

type BatchedRepositoryPublisherParams struct {
	dig.In

	Cfg    *config.Config
	Repo   Repository
	Logger *zap.Logger
}

func BatchedRepositoryPublisher(p BatchedRepositoryPublisherParams) EventPublisher {
	adder := make(chan Event, 5000)

	go func() {
		accumulated := make([]Event, 0, 5000)
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
							zap.Error(err))
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

func (ep *batchedRepositoryPublisher) PublishEvent(evt RawEvent, timeLayout string, fileParser parser.FileParser) {
	processed := Event{
		Raw:      evt.Raw,
		Host:     ep.cfg.HostName,
		SourceId: evt.SourceId,
		Source:   evt.Source,
		Offset:   evt.Offset,
	}

	fields, err := parser.ExtractFields(strings.ToLower(evt.Raw), fileParser)
	if err != nil {
		ep.logger.Warn("failed to extract fields when getting timestamp, will use current time as timestamp",
			zap.Error(err))
		processed.Timestamp = time.Now()
	} else if t, ok := fields["_time"]; ok {
		parsed, err := parser.ParseTime(timeLayout, t)
		if err != nil {
			ep.logger.Warn("failed to parse _time field, will use current time as timestamp",
				zap.Error(err))
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

type nopEventPublisher struct {
}

func NopEventPublisher() EventPublisher {
	return &nopEventPublisher{}
}

func (ep *nopEventPublisher) PublishEvent(_ RawEvent, _ string, _ parser.FileParser) {}
