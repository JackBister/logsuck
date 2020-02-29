package parser

import (
	"errors"
	"fmt"
	"github.com/jackbister/logsuck/internal/config"
)

type ParseResult struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string]string
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

func ExtractFields(input string, cfg *config.Config) map[string]string {
	ret := map[string]string{}
	for _, rex := range cfg.FieldExtractors {
		matches := rex.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if len(match) == 2 && len(rex.SubexpNames()) == 2 {
				ret[rex.SubexpNames()[1]] = match[0]
			} else if len(matches) > 2 {
				ret[match[1]] = match[2]
			}
		}
	}
	return ret
}

func Parse(input string) (*ParseResult, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("error while tokenizing: %w", err)
	}

	p := parser{
		tokens: tokens,
	}

	ret := ParseResult{
		Fragments:    map[string]struct{}{},
		NotFragments: map[string]struct{}{},
		Fields:       map[string]string{},
	}

	for len(p.tokens) > 0 {
		tok := p.take()
		if tok == nil {
			break
		}
		if tok.typ == tokenString {
			if p.peek() == tokenEquals {
				p.take()
				if p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("unexpected token, expected string or quoted string after =")
				}
				value := p.take()
				ret.Fields[tok.value] = value.value
			} else {
				ret.Fragments[tok.value] = struct{}{}
			}
		} else if tok.typ == tokenQuotedString {
			ret.Fragments[tok.value] = struct{}{}
		} else if tok.typ == tokenKeyword {
			if tok.value == "NOT" {
				if p.peek() == tokenWhitespace {
					p.take()
				}
				if p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("unexpected token, expected string or quoted string after =")
				}
				frag := p.take().value
				ret.NotFragments[frag] = struct{}{}
			}
		}
	}

	return &ret, nil
}
