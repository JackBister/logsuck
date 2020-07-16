package events

import "time"

// RawEvent represents an Event that has not yet been enriched with information about field values etc.
type RawEvent struct {
	Raw    string
	Source string
	Offset int64
}

type Event struct {
	Raw       string
	Timestamp time.Time
	Source    string
	Offset    int64
}

type EventWithId struct {
	Id        int64
	Raw       string
	Timestamp time.Time
	Source    string
}

type EventWithExtractedFields struct {
	Id        int64
	Raw       string
	Timestamp time.Time
	Source    string
	Fields    map[string]string
}

type EventIdAndTimestamp struct {
	Id        int64
	Timestamp time.Time
}
