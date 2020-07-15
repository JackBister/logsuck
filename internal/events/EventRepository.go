package events

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/filtering"
)

type Repository interface {
	Add(evt Event) (id *int64, err error)
	AddBatch(events []Event) ([]int64, error)
	FilterStream(sources, notSources map[string]struct{}, fragments map[string]struct{}, startTime, endTime *time.Time) <-chan []EventWithId
	GetByIds(ids []int64) ([]EventWithId, error)
}

type inMemoryRepository struct {
	events map[int64]EventWithId
}

func InMemoryRepository() Repository {
	return &inMemoryRepository{
		events: map[int64]EventWithId{},
	}
}

func (repo *inMemoryRepository) Add(evt Event) (*int64, error) {
	// TODO: thread safety
	id := int64(len(repo.events))
	repo.events[id] = EventWithId{
		Id:        id,
		Raw:       evt.Raw,
		Timestamp: evt.Timestamp,
		Source:    evt.Source,
	}
	return &id, nil
}

func (repo *inMemoryRepository) AddBatch(evts []Event) ([]int64, error) {
	ret := make([]int64, len(evts))
	for i, evt := range evts {
		id, err := repo.Add(evt)
		if err != nil {
			// Can't actually happen?
			return nil, fmt.Errorf("error when adding batch: %w", err)
		}
		ret[i] = *id
	}
	return ret, nil
}

func (repo *inMemoryRepository) FilterStream(sources, notSources map[string]struct{}, fragments map[string]struct{}, startTime, endTime *time.Time) <-chan []EventWithId {
	ret := make(chan []EventWithId)
	go func() {
		compiledSources := filtering.CompileKeys(sources)
		compiledNotSources := filtering.CompileKeys(notSources)
		for _, evt := range repo.events {
			if shouldIncludeEvent(&evt, compiledSources, compiledNotSources, startTime, endTime) {
				ret <- []EventWithId{evt}
			}
		}
		close(ret)
	}()
	return ret
}

func shouldIncludeEvent(evt *EventWithId, compiledSources, compiledNotSources []*regexp.Regexp, startTime, endTime *time.Time) bool {
	if startTime != nil && evt.Timestamp.Before(*startTime) {
		return false
	}
	if endTime != nil && evt.Timestamp.After(*endTime) {
		return false
	}
	include := false
	for _, rex := range compiledSources {
		if rex.MatchString(evt.Source) {
			include = true
			break
		}
	}
	exclude := false
	for _, rex := range compiledNotSources {
		if rex.MatchString(evt.Source) {
			exclude = true
			break
		}
	}
	return (len(compiledSources) == 0 && !exclude) || (include && !exclude)
}

func (repo *inMemoryRepository) GetByIds(ids []int64) ([]EventWithId, error) {
	ret := make([]EventWithId, 0, len(ids))
	missingIds := make([]string, 0)

	for _, id := range ids {
		if evt, ok := repo.events[id]; !ok {
			missingIds = append(missingIds, string(id))
		} else {
			ret = append(ret, evt)
		}
	}

	if len(missingIds) > 0 {
		return nil, errors.New("did not find events with ids=[" + strings.Join(missingIds, ", ") + "]")
	}

	return ret, nil
}
