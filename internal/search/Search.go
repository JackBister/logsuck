package search

import (
	"github.com/jackbister/logsuck/internal/parser"
	"github.com/pkg/errors"
)

type Search struct {
	Fragments map[string]struct{}
	Fields    map[string]string
}

func Parse(searchString string) (*Search, error) {
	res, err := parser.Parse(searchString, parser.ParseModeSearch)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing")
	}

	ret := Search{
		Fragments: res.Fragments,
		Fields:    res.Fields,
	}

	return &ret, nil
}
