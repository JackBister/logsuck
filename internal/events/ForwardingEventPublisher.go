package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/jackbister/logsuck/internal/config"
)

const forwardChunkSize = 1000

type forwardingEventPublisher struct {
	cfg *config.Config

	accumulated []RawEvent
	adder       chan<- RawEvent
}

func ForwardingEventPublisher(cfg *config.Config) EventPublisher {
	adder := make(chan RawEvent)
	ep := forwardingEventPublisher{
		cfg: cfg,

		accumulated: make([]RawEvent, 0, forwardChunkSize),
		adder:       adder,
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
						log.Println("error when adding events:", err)
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
						log.Println("error when adding events:", err)
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

func (ep *forwardingEventPublisher) PublishEvent(evt RawEvent, timeLayout string) {
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
		req := receiveEventsRequest{
			Events: evts,
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
		log.Printf("forwarded numEvents=%v in timeInMs=%v\n", len(evts), time.Now().Sub(startTime).Milliseconds())
		ep.accumulated = ep.accumulated[chunkSize:]
	}
	return nil
}

func (ep *forwardingEventPublisher) dropExcessEvents() {
	if len(ep.accumulated) > ep.cfg.Forwarder.MaxBufferedEvents {
		log.Printf("number of buffered events exceeded maxBufferedEvents=%v, will drop events to keep buffer size down.\n", ep.cfg.Forwarder.MaxBufferedEvents)
		numOver := len(ep.accumulated) - ep.cfg.Forwarder.MaxBufferedEvents
		ep.accumulated = ep.accumulated[numOver:] // TODO: Is the GC actually able to free the memory of ep.accumulated[:numOver] here? I'm assuming it will if the slice is reallocated later due to appending?
	} else if quota := float64(len(ep.accumulated)) / float64(ep.cfg.Forwarder.MaxBufferedEvents); quota > 0.7 {
		log.Printf("warning: number of buffered events is %.2f %% of maxBufferedEvents (%v/%v). If maxBufferedEvents is exceeded events will be lost. "+
			"This may indicate a connection problem or the recipient instance is not running.\n",
			quota*100, len(ep.accumulated), ep.cfg.Forwarder.MaxBufferedEvents)
	}
}
