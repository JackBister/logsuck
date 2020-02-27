package parser

import (
	"errors"
	errors2 "github.com/pkg/errors"
)

type ParseMode int

const (
	ParseModeIngest ParseMode = 0
	ParseModeSearch           = 1
)

type ParseResult struct {
	Fragments map[string]struct{}
	Fields    map[string]string
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

func Parse(input string, mode ParseMode) (*ParseResult, error) {
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

	for len(p.tokens) > 0 {
		tok := p.take()
		if tok == nil {
			break
		}
		if tok.typ == tokenString {
			if p.peek() == tokenEquals {
				p.take()
				if mode != ParseModeSearch && p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("Unexpected token=" + string(p.peek()) + ", expected string or quoted string after =")
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
