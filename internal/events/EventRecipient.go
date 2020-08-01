package events

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jackbister/logsuck/internal/config"
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
		_, err = er.repo.AddBatch(req.Events)
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
	Events []Event
}
