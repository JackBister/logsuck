// Copyright 2020 The Logsuck Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"errors"
	"fmt"
	"strings"
)

type SearchParseResult struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
	Hosts        map[string]struct{}
	NotHosts     map[string]struct{}
}

func ParseSearch(input string) (*SearchParseResult, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("error while tokenizing: %w", err)
	}

	p := parser{
		tokens: tokens,
	}

	ret := SearchParseResult{
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
