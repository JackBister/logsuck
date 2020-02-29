package events

import "time"

// RawEvent represents an Event that has not yet been enriched with information about field values etc.
type RawEvent struct {
	Raw    string
	Source string
}

type Event struct {
	Raw       string
	Timestamp time.Time
	Source    string
}

type EventWithExtractedFields struct {
	Raw       string
	Timestamp time.Time
	Source    string
	Fields    map[string]string
}
