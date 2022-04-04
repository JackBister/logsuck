package rpc

import "time"

type RawEvent struct {
	Raw      string
	Host     string
	Source   string
	SourceId string
	Offset   int64
}

type ReceiveEventsRequest struct {
	HostType string
	Events   []RawEvent
}

type ConfigResponse struct {
	LastUpdateTime *time.Time
	Config         map[string]string
}
