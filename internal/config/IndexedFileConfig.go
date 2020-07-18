package config

import (
	"regexp"
	"time"
)

// IndexedFileConfig contains configuration for a specific file which will be indexed
type IndexedFileConfig struct {
	// Filename is the name of the file. This will be used to set the "source" field of the event.
	Filename string
	// EventDelimiter is a regex that is used to determine where one event ends and another begins.
	// The default is "\n".
	EventDelimiter *regexp.Regexp
	// ReadInterval is the time the file watcher will sleep between looking for new events in the file.
	// A lower duration will make events arrive faster in the search engine, but will consume more CPU.
	// The default is 10 * time.Second.
	ReadInterval time.Duration
	// TimeLayout is the layout of the _time field if it is extracted, following Go's time.Parse style https://golang.org/pkg/time/#Parse
	// The default is "2006/01/02 15:04:05"
	TimeLayout string
}
