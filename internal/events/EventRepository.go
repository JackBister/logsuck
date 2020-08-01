package events

import "github.com/jackbister/logsuck/internal/search"

type SortMode = int

const (
	SortModeNone          SortMode = 0
	SortModeTimestampDesc SortMode = 1
)

type Repository interface {
	AddBatch(events []Event) ([]int64, error)
	FilterStream(srch *search.Search) <-chan []EventWithId
	GetByIds(ids []int64, sortMode SortMode) ([]EventWithId, error)
}
