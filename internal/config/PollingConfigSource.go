package config

import (
	"log"
	"time"
)

type PollingConfigSource struct {
	changes        chan struct{}
	cs             PollableConfigSource
	interval       time.Duration
	lastSeenUpdate *time.Time
}

func NewPollingConfigSource(cs PollableConfigSource, interval time.Duration) *PollingConfigSource {
	changes := make(chan struct{})
	t, err := cs.GetLastUpdateTime()
	if err != nil {
		log.Printf("got error when getting last update time from pollable config source: %v\n", err)

	}
	ret := &PollingConfigSource{
		changes:        changes,
		cs:             cs,
		interval:       interval,
		lastSeenUpdate: t,
	}
	go func() {
		ticker := time.NewTicker(interval)
		for {
			<-ticker.C
			t, err := cs.GetLastUpdateTime()
			if err != nil {
				log.Printf("got error when polling config source: %v\n", err)
				continue
			}
			if t == nil {
				log.Println("got nil lastUpdateTime when polling config source")
				continue
			}
			if ret.lastSeenUpdate == nil || t.After(*ret.lastSeenUpdate) {
				log.Println("config change detected, sending change event")
				ret.changes <- struct{}{}
			}
		}
	}()

	return ret
}

func (p *PollingConfigSource) Changes() <-chan struct{} {
	return p.changes
}

func (p *PollingConfigSource) Get(name string) (string, bool) {
	return p.cs.Get(name)
}

func (p *PollingConfigSource) GetKeys() ([]string, bool) {
	return p.cs.GetKeys()
}
