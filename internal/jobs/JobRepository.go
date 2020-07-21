package jobs

import (
	"time"

	"github.com/jackbister/logsuck/internal/events"
)

type Repository interface {
	AddResults(id int64, events []events.EventIdAndTimestamp) error
	AddFieldStats(id int64, fields []FieldStats) error
	Get(id int64) (*Job, error)
	GetResults(id int64, skip int, take int) (eventIds []int64, err error)
	GetFieldOccurences(id int64) (map[string]int, error)
	GetFieldValues(id int64, fieldName string) (map[string]int, error)
	GetNumMatchedEvents(id int64) (int64, error)
	Insert(query string, startTime, endTime *time.Time) (id *int64, err error)
	UpdateState(id int64, state JobState) error
}

type FieldStats struct {
	Key         string
	Value       string
	Occurrences int
}
