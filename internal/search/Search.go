package search

import (
	"fmt"
	"time"

	"github.com/jackbister/logsuck/internal/parser"
)

type Search struct {
	StartTime, EndTime *time.Time

	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
	Hosts        map[string]struct{}
	NotHosts     map[string]struct{}
}

func Parse(searchString string, startTime, endTime *time.Time) (*Search, error) {
	res, err := parser.Parse(searchString)
	if err != nil {
		return nil, fmt.Errorf("error while parsing: %w", err)
	}

	ret := Search{
		StartTime: startTime,
		EndTime:   endTime,

		Fragments:    res.Fragments,
		NotFragments: res.NotFragments,
		Fields:       res.Fields,
		NotFields:    res.NotFields,
		Sources:      res.Sources,
		NotSources:   res.NotSources,
		Hosts:        res.Hosts,
		NotHosts:     res.NotHosts,
	}

	return &ret, nil
}
