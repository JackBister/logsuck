package search

import (
	"fmt"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/filtering"
	"github.com/jackbister/logsuck/internal/parser"
	"strings"
)

type Search struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
}

func Parse(searchString string) (*Search, error) {
	res, err := parser.Parse(searchString)
	if err != nil {
		return nil, fmt.Errorf("error while parsing: %w", err)
	}

	ret := Search{
		Fragments:    res.Fragments,
		NotFragments: res.NotFragments,
		Fields:       res.Fields,
		NotFields:    res.NotFields,
		Sources:      res.Sources,
		NotSources:   res.NotSources,
	}

	return &ret, nil
}

// SearchEvents searches a slice of events based on a parsed Search object.
// It is the callers responsibility to filter by source and time, which should be done before calling SearchEvents since
// it performs some heavyweight operations
func FilterEvents(repo events.Repository, srch *Search, cfg *config.Config) []events.EventWithExtractedFields {
	inputEvents := repo.Filter(srch.Sources, srch.NotSources)
	ret := make([]events.EventWithExtractedFields, 0, 1)
	compiledFrags := filtering.CompileKeys(srch.Fragments)
	compiledNotFrags := filtering.CompileKeys(srch.NotFragments)
	compiledFields := filtering.CompileMap(srch.Fields)
	compiledNotFields := filtering.CompileMap(srch.NotFields)
	for _, evt := range inputEvents {
		rawLowered := strings.ToLower(evt.Raw)
		evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), cfg.FieldExtractors)
		// TODO: This could produce unexpected results
		evtFields["source"] = evt.Source

		include := true
		for _, frag := range compiledFrags {
			if !frag.MatchString(rawLowered) {
				include = false
				break
			}
		}
		for _, frag := range compiledNotFrags {
			if frag.MatchString(rawLowered) {
				include = false
				break
			}
		}
		for key, values := range compiledFields {
			evtValue, ok := evtFields[key]
			if !ok {
				include = false
				break
			}
			anyMatch := false
			for _, value := range values {
				if value.MatchString(evtValue) {
					anyMatch = true
				}
			}
			if !anyMatch {
				include = false
				break
			}
		}
		for key, values := range compiledNotFields {
			evtValue, ok := evtFields[key]
			if !ok {
				break
			}
			anyMatch := false
			for _, value := range values {
				if value.MatchString(evtValue) {
					anyMatch = true
				}
			}
			if anyMatch {
				include = false
				break
			}
		}

		if include {
			ret = append(ret, events.EventWithExtractedFields{
				Raw:       evt.Raw,
				Timestamp: evt.Timestamp,
				Source:    evt.Source,
				Fields:    evtFields,
			})
		}
	}
	return ret
}
