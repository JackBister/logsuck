package parser

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
)

type ParseResult struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
	Hosts        map[string]struct{}
	NotHosts     map[string]struct{}
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

func (p *parser) peekValue() string {
	if len(p.tokens) == 0 {
		return ""
	}
	return p.tokens[0].value
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

func (p *parser) parseParenList() ([]string, error) {
	ret := make([]string, 0)
	if p.peek() != tokenLparen {
		return nil, errors.New("unexpected token, expected '(' after 'IN'")
	}
	p.take()
	p.skipWhitespace()
	for p.peek() == tokenString || p.peek() == tokenQuotedString {
		tok := p.take()
		ret = append(ret, tok.value)
		p.skipWhitespace()
		if p.peek() != tokenComma && p.peek() != tokenRparen {
			return nil, errors.New("unexpected token, expected ',' or ')' after string in parenthesis list")
		}
		if p.peek() == tokenRparen {
			break
		}
		p.take()
		p.skipWhitespace()
		if p.peek() != tokenString && p.peek() != tokenQuotedString {
			return nil, errors.New("unexpected token, expected string after comma in parenthesis list")
		}
	}
	p.skipWhitespace()
	if p.peek() != tokenRparen {
		return nil, errors.New("unexpected token, expected ')' at end of IN expression")
	}
	p.take()
	return ret, nil
}

func (p *parser) skipWhitespace() {
	for p.tokens[0].typ == tokenWhitespace {
		p.tokens = p.tokens[1:]
	}
}

func (p *parser) take() *token {
	ret := &p.tokens[0]
	p.tokens = p.tokens[1:]
	return ret
}

func ExtractFields(input string, fieldExtractors []*regexp.Regexp) map[string]string {
	ret := map[string]string{}
	for _, rex := range fieldExtractors {
		subExpNames := rex.SubexpNames()[1:]
		isNamedOnlyExtractor := true
		for _, name := range subExpNames {
			if name == "" {
				isNamedOnlyExtractor = false
			}
		}
		matches := rex.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if isNamedOnlyExtractor && len(rex.SubexpNames()) == len(match) {
				for j, name := range subExpNames {
					ret[name] = match[j+1]
				}
			} else if len(match) == 3 {
				ret[match[1]] = match[2]
			} else {
				log.Printf("Malformed field extractor '%v': If there are any unnamed capture groups in the regex, there must be exactly two capture groups.\n", rex)
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
		Fields:       map[string][]string{},
		NotFields:    map[string][]string{},
		Sources:      map[string]struct{}{},
	}

	for len(p.tokens) > 0 {
		tok := p.take()
		if tok == nil {
			break
		}
		if tok.typ == tokenString {
			lowered := strings.ToLower(tok.value)
			if p.peek() == tokenEquals {
				p.take()
				if p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("unexpected token, expected string or quoted string after =")
				}
				value := p.take()
				ret.Fields[lowered] = []string{value.value}
			} else if p.peek() == tokenNotEquals {
				p.take()
				if p.peek() != tokenString && p.peek() != tokenQuotedString {
					return nil, errors.New("unexpected token, expected string or quoted string after =")
				}
				value := p.take()
				if existingNots, ok := ret.NotFields[lowered]; ok {
					ret.NotFields[lowered] = append(existingNots, value.value)
				} else {
					ret.NotFields[lowered] = []string{value.value}
				}
			} else if p.peek() == tokenWhitespace {
				p.skipWhitespace()
				if p.peek() == tokenKeyword && p.peekValue() == "IN" {
					p.take()
					p.skipWhitespace()
					values, err := p.parseParenList()
					if err != nil {
						return nil, fmt.Errorf("error while parsing IN expression: %w", err)
					}
					ret.Fields[lowered] = values
				} else if p.peek() == tokenKeyword && p.peekValue() == "NOT" {
					p.take()
					p.skipWhitespace()
					if p.peek() != tokenKeyword || p.peekValue() != "IN" {
						return nil, errors.New("unexpected token, expected 'IN' after 'NOT'")
					}
					p.take()
					p.skipWhitespace()
					values, err := p.parseParenList()
					if err != nil {
						return nil, fmt.Errorf("error while parsing NOT IN expression: %w", err)
					}
					if existingNots, ok := ret.NotFields[lowered]; ok {
						ret.NotFields[lowered] = append(existingNots, values...)
					} else {
						ret.NotFields[lowered] = values
					}
				} else {
					ret.Fragments[tok.value] = struct{}{}
				}
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

	if sources, ok := ret.Fields["source"]; ok {
		ret.Sources = make(map[string]struct{}, len(sources))
		for _, src := range sources {
			ret.Sources[src] = struct{}{}
		}
	}
	if sources, ok := ret.NotFields["source"]; ok {
		ret.NotSources = make(map[string]struct{}, len(sources))
		for _, src := range sources {
			ret.NotSources[src] = struct{}{}
		}
	}
	if hosts, ok := ret.Fields["host"]; ok {
		ret.Hosts = make(map[string]struct{}, len(hosts))
		for _, host := range hosts {
			ret.Hosts[host] = struct{}{}
		}
	}
	if hosts, ok := ret.NotFields["host"]; ok {
		ret.NotHosts = make(map[string]struct{}, len(hosts))
		for _, host := range hosts {
			ret.NotHosts[host] = struct{}{}
		}
	}

	return &ret, nil
}
