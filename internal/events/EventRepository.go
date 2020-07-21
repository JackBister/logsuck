package events

import (
	"time"
)

type SortMode = int

const (
	SortModeNone          SortMode = 0
	SortModeTimestampDesc SortMode = 1
)

type Repository interface {
	AddBatch(events []Event) ([]int64, error)
	FilterStream(sources, notSources map[string]struct{}, fragments map[string]struct{}, startTime, endTime *time.Time) <-chan []EventWithId
	GetByIds(ids []int64, sortMode SortMode) ([]EventWithId, error)
}
