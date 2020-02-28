package search

import (
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
	"github.com/pkg/errors"
	"strings"
)

type Search struct {
	Fragments map[string]struct{}
	Fields    map[string]string
}

func Parse(searchString string, cfg *config.Config) (*Search, error) {
	res, err := parser.Parse(strings.ToLower(searchString), parser.ParseModeSearch, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing")
	}

	ret := Search{
		Fragments: res.Fragments,
		Fields:    res.Fields,
	}

	return &ret, nil
}
