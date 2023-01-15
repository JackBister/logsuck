package rpc

import (
	"time"

	"github.com/jackbister/logsuck/internal/config"
)

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
	Modified *time.Time
	Config   config.JsonConfig
}
