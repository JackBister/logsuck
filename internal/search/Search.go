package search

import (
	"fmt"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/parser"
	"strings"
)

type Search struct {
	Fragments map[string]struct{}
	Fields    map[string]string
}

func Parse(searchString string, cfg *config.Config) (*Search, error) {
	res, err := parser.Parse(strings.ToLower(searchString), parser.ParseModeSearch, cfg)
	if err != nil {
		return nil, fmt.Errorf("error while parsing: %w", err)
	}

	ret := Search{
		Fragments: res.Fragments,
		Fields:    res.Fields,
	}

	return &ret, nil
}
