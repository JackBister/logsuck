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

package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
	"github.com/jackbister/logsuck/internal/rpc"
	"go.uber.org/zap"
)

const forwardChunkSize = 1000

type forwardingEventPublisher struct {
	cfg *config.Config

	accumulated []events.RawEvent
	adder       chan<- events.RawEvent

	logger *zap.Logger
}

func ForwardingEventPublisher(cfg *config.Config, logger *zap.Logger) events.EventPublisher {
	adder := make(chan events.RawEvent)
	ep := forwardingEventPublisher{
		cfg: cfg,

		accumulated: make([]events.RawEvent, 0, forwardChunkSize),
		adder:       adder,

		logger: logger,
	}

	go func() {
		lastErrorTime := time.Now().Add(-5 * time.Second)
		timeout := time.After(1 * time.Second)
		for {
			now := time.Now()
			select {
			case <-timeout:
				if len(ep.accumulated) > 0 {
					err := ep.forward()
					if err != nil {
						logger.Error("error when adding events",
							zap.Error(err))
						lastErrorTime = time.Now()
					}
					ep.dropExcessEvents()
				}
				timeout = time.After(1 * time.Second)
			case evt := <-adder:
				ep.accumulated = append(ep.accumulated, evt)
				if len(ep.accumulated) >= forwardChunkSize && now.Sub(lastErrorTime).Seconds() > 1 {
					err := ep.forward()
					if err != nil {
						logger.Error("error when adding events",
							zap.Error(err))
						lastErrorTime = time.Now()
					}
					ep.dropExcessEvents()
					timeout = time.After(1 * time.Second)
				}
			}
		}
	}()

	return &ep
}

func (ep *forwardingEventPublisher) PublishEvent(evt events.RawEvent, timeLayout string, fileParser parser.FileParser) {
	ep.adder <- evt
}

func (ep *forwardingEventPublisher) forward() error {
	for len(ep.accumulated) > 0 {
		startTime := time.Now()
		chunkSize := forwardChunkSize
		if len(ep.accumulated) < forwardChunkSize {
			chunkSize = len(ep.accumulated)
		}
		evts := ep.accumulated[:chunkSize]
		req := rpc.ReceiveEventsRequest{
			HostType: ep.cfg.HostType,
			Events:   toRpcEvents(evts),
		}
		serialized, err := json.Marshal(req)
		if err != nil {
			ep.accumulated = ep.accumulated[chunkSize:]
			return fmt.Errorf("failed to serialize events for forwarding. Events will not be buffered: %w", err)
		}
		resp, err := http.Post(ep.cfg.Forwarder.RecipientAddress+"/v1/receiveEvents", "application/json", bytes.NewReader(serialized))
		if err != nil {
			return fmt.Errorf("failed to forward events. Events will be buffered: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			bodyString := ""
			if err == nil {
				bodyString = string(bodyBytes)
			}
			return fmt.Errorf("failed to forward events: got non-200 statusCode=%v, body='%v'. Events will be buffered", resp.StatusCode, bodyString)
		}
		ep.logger.Info("finished forwarding events",
			zap.Int("numEvents", len(evts)),
			zap.Stringer("duration", time.Now().Sub(startTime)))
		ep.accumulated = ep.accumulated[chunkSize:]
	}
	return nil
}

func (ep *forwardingEventPublisher) dropExcessEvents() {
	if len(ep.accumulated) > ep.cfg.Forwarder.MaxBufferedEvents {
		ep.logger.Error("number of buffered events exceeded maxBufferedEvents, will drop events to keep buffer size down.",
			zap.Int("maxBufferedEvents", ep.cfg.Forwarder.MaxBufferedEvents))
		numOver := len(ep.accumulated) - ep.cfg.Forwarder.MaxBufferedEvents
		ep.accumulated = ep.accumulated[numOver:] // TODO: Is the GC actually able to free the memory of ep.accumulated[:numOver] here? I'm assuming it will if the slice is reallocated later due to appending?
	} else if quota := float64(len(ep.accumulated)) / float64(ep.cfg.Forwarder.MaxBufferedEvents); quota > 0.7 {
		ep.logger.Warn("number of buffered events is nearing maxBufferedEvents. If maxBufferedEvents is exceeded events will be lost. "+
			"This may indicate a connection problem or the recipient instance is not running.",
			zap.Float64("percentageFull", quota*100),
			zap.Int("accumulatedEvents", len(ep.accumulated)),
			zap.Int("maxBufferedEvents", ep.cfg.Forwarder.MaxBufferedEvents))
	}
}

func toRpcEvents(evts []events.RawEvent) []rpc.RawEvent {
	ret := make([]rpc.RawEvent, len(evts))
	for i, e := range evts {
		ret[i] = rpc.RawEvent{
			Raw:      e.Raw,
			Host:     e.Host,
			Source:   e.Source,
			SourceId: e.SourceId,
			Offset:   e.Offset,
		}
	}
	return ret
}
