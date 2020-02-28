package parser

import (
	"errors"
	"github.com/jackbister/logsuck/internal/config"
	errors2 "github.com/pkg/errors"
	"time"
)

type ParseMode int

const (
	ParseModeIngest ParseMode = 0
	ParseModeSearch           = 1
)

type ParseResult struct {
	Fragments map[string]struct{}
	Fields    map[string]string
	Time      time.Time
}

type parser struct {
	tokens []token
}

func (p *parser) peek() tokenType {
	if len(p.tokens) == 0 {
		return tokenInvalid
	}
	return p.tokens[0].typ
}

func (p *parser) require(expected tokenType) (*token, error) {
	if len(p.tokens) == 0 {
		return nil, errors.New("Unexpected end of string, expected tokenType=" + string(expected))
	}
	if p.tokens[0].typ != expected {
		return nil, errors.New("Unexpected tokenType=" + string(p.tokens[0].typ) + ", expected tokenType=" + string(expected))
	}
	ret := &p.tokens[0]
	p.tokens = p.tokens[1:]
	return ret, nil
}

func (p *parser) take() *token {
	ret := &p.tokens[0]
	p.tokens = p.tokens[1:]
	return ret
}

func Parse(input string, mode ParseMode, cfg *config.Config) (*ParseResult, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, errors2.Wrap(err, "error while tokenizing")
	}

	p := parser{
		tokens: tokens,
	}

	ret := ParseResult{
		Fragments: map[string]struct{}{},
		Fields:    map[string]string{},
	}

	for _, rex := range cfg.FieldExtractors {
		matches := rex.FindStringSubmatch(input)
		if len(matches) == 2 && len(rex.SubexpNames()) == 2 {
			ret.Fields[rex.SubexpNames()[1]] = matches[0]
		} else if len(matches) > 2 {
			ret.Fields[matches[1]] = matches[2]
		}
	}

	if _, ok := ret.Fields["_time"]; !ok && mode == ParseModeIngest {
		ret.Fields["_time"] = time.Now().Format(cfg.TimeLayout)
	}

	ret.Time, err = time.Parse(cfg.TimeLayout, ret.Fields["_time"])
	if err != nil {
		ret.Time = time.Now()
	}

	for len(p.tokens) > 0 {
		tok := p.take()
		if tok == nil {
			break
		}
		if tok.typ == tokenString {
			if p.peek() == tokenEquals {
				p.take()
				if mode == ParseModeSearch && p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("unexpected token, expected string or quoted string after =")
				}
				value := p.take()
				ret.Fields[tok.value] = value.value
			} else {
				ret.Fragments[tok.value] = struct{}{}
			}
		}
	}

	return &ret, nil
}
