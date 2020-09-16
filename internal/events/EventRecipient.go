// Copyright 2020 The Logsuck Authors
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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
)

type EventRecipient struct {
	cfg  *config.Config
	repo Repository
}

func NewEventRecipient(cfg *config.Config, repo Repository) *EventRecipient {
	return &EventRecipient{cfg: cfg, repo: repo}
}

func (er *EventRecipient) Serve() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/receiveEvents", func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "no body: body must be a JSON encoded object", 400)
			return
		}
		defer r.Body.Close()
		if r.Method != "POST" {
			http.Error(w, "unsupported method: must be POST", 405)
			return
		}
		var req receiveEventsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to decode JSON: %v", err), 500)
			return
		}
		processed := make([]Event, len(req.Events))
		for i, evt := range req.Events {
			processed[i] = Event{
				Raw:    evt.Raw,
				Host:   evt.Host,
				Source: evt.Source,
				Offset: evt.Offset,
			}

			var timeLayout string
			if tl, ok := er.cfg.Recipient.TimeLayouts[evt.Source]; ok {
				timeLayout = tl
			} else {
				timeLayout = er.cfg.Recipient.TimeLayouts["DEFAULT"]
			}

			fields := parser.ExtractFields(strings.ToLower(evt.Raw), er.cfg.FieldExtractors)
			if t, ok := fields["_time"]; ok {
				parsed, err := time.Parse(timeLayout, t)
				if err != nil {
					log.Printf("failed to parse _time field, will use current time as timestamp: %v\n", err)
					processed[i].Timestamp = time.Now()
				} else {
					processed[i].Timestamp = parsed
				}
			} else {
				log.Println("no _time field extracted, will use current time as timestamp")
				processed[i].Timestamp = time.Now()
			}
		}
		_, err = er.repo.AddBatch(processed)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to add events to repository: %v", err), 500)
			return
		}
	})

	s := &http.Server{
		Addr:    er.cfg.Recipient.Address,
		Handler: mux,
	}

	log.Printf("Starting EventRecipient on address='%v'\n", er.cfg.Recipient.Address)
	return s.ListenAndServe()
}

type receiveEventsRequest struct {
	Events []RawEvent
}
