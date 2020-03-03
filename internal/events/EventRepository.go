package events

import (
	"github.com/jackbister/logsuck/internal/filtering"
	"time"
)

type Repository interface {
	Add(evt Event)
	Filter(sources, notSources map[string]struct{}, startTime, endTime *time.Time) []Event
}

type inMemoryRepository struct {
	events []Event
}

func InMemoryRepository() Repository {
	return &inMemoryRepository{
		events: make([]Event, 0),
	}
}

func (repo *inMemoryRepository) Add(evt Event) {
	// TODO: thread safety
	repo.events = append(repo.events, evt)
}

func (repo *inMemoryRepository) Filter(sources, notSources map[string]struct{}, startTime, endTime *time.Time) []Event {
	ret := make([]Event, 0)
	// TODO you can shortcut here if not using wildcards - need to measure if this is useful
	compiledSources := filtering.CompileKeys(sources)
	compiledNotSources := filtering.CompileKeys(notSources)
	/*
		startIdx := sort.Search(len(repo.events), func(i int) bool {
			if startTime != nil && repo.events[i].Timestamp.Before(*startTime) {
				return false
			}
			if endTime != nil && repo.events[i].Timestamp.After(*endTime) {
				return false
			}
			return true
		})
	*/
	for _, evt := range repo.events {
		if startTime != nil && evt.Timestamp.Before(*startTime) {
			continue
		}
		if endTime != nil && evt.Timestamp.After(*endTime) {
			continue
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
		if (len(sources) == 0 && !exclude) || (include && !exclude) {
			ret = append(ret, evt)
		}
	}
	return ret
}
