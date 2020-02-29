package search

import (
	"fmt"
	"github.com/jackbister/logsuck/internal/parser"
)

type Search struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
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
	}

	return &ret, nil
}
