package events

import (
	"strings"

	"github.com/jackbister/logsuck/internal/search"
)

type Repository interface {
	Add(evt Event)
	Search(srch *search.Search) []Event
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

func (repo *inMemoryRepository) Search(srch *search.Search) []Event {
	ret := make([]Event, 0, 1)
	loweredFrags := lowerFrags(srch)
	for _, evt := range repo.events {
		rawLowered := strings.ToLower(evt.Raw)

		include := true
		for _, frag := range loweredFrags {
			if !strings.Contains(rawLowered, frag) {
				include = false
			}
		}
		for key, value := range srch.Fields {
			if evtValue, ok := evt.Fields[key]; !ok || evtValue != value {
				include = false
			}
		}

		if include {
			ret = append(ret, evt)
		}
	}
	return ret
}

func lowerFrags(srch *search.Search) []string {
	loweredFrags := make([]string, len(srch.Fragments))
	i := 0
	for frag := range srch.Fragments {
		loweredFrags[i] = strings.ToLower(frag)
		i++
	}
	return loweredFrags
}
