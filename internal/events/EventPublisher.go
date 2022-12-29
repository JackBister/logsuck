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
	"log"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
)

type EventPublisher interface {
	PublishEvent(evt RawEvent, timeLayout string, fileParser parser.FileParser)
}

type batchedRepositoryPublisher struct {
	cfg  *config.Config
	repo Repository

	adder chan<- Event
}

func BatchedRepositoryPublisher(cfg *config.Config, repo Repository) EventPublisher {
	adder := make(chan Event, 5000)

	go func() {
		accumulated := make([]Event, 0, 5000)
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
				if len(accumulated) >= 5000 {
					err := repo.AddBatch(accumulated)
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

func (ep *batchedRepositoryPublisher) PublishEvent(evt RawEvent, timeLayout string, fileParser parser.FileParser) {
	processed := Event{
		Raw:      evt.Raw,
		Host:     ep.cfg.HostName,
		SourceId: evt.SourceId,
		Source:   evt.Source,
		Offset:   evt.Offset,
	}

	fields := parser.ExtractFields(strings.ToLower(evt.Raw), fileParser)
	if t, ok := fields["_time"]; ok {
		parsed, err := time.Parse(timeLayout, t)
		if err != nil {
			log.Printf("failed to parse _time field, will use current time as timestamp: %v\n", err)
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

type debugEventPublisher struct {
	wrapped EventPublisher
}

func DebugEventPublisher(wrapped EventPublisher) EventPublisher {
	return &debugEventPublisher{
		wrapped: wrapped,
	}
}

func (ep *debugEventPublisher) PublishEvent(evt RawEvent, timeLayout string, fileParser parser.FileParser) {
	log.Println("Received event:", evt)
	if ep.wrapped != nil {
		ep.wrapped.PublishEvent(evt, timeLayout, fileParser)
	}
}

type nopEventPublisher struct {
}

func NopEventPublisher() EventPublisher {
	return &nopEventPublisher{}
}

func (ep *nopEventPublisher) PublishEvent(_ RawEvent, _ string, _ parser.FileParser) {}
