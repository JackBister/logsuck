// Copyright 2021 Jack Bister
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

	"github.com/jackbister/logsuck/pkg/logsuck/search"
)

func Parse(searchString string) (*search.Search, error) {
	res, err := ParseSearch(searchString)
	if err != nil {
		return nil, fmt.Errorf("error while parsing: %w", err)
	}

	ret := search.Search{
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
		return nil, errors.New("Unexpected end of string, expected tokenType=" + fmt.Sprint(expected))
	}
	if p.tokens[0].typ != expected {
		return nil, errors.New("Unexpected tokenType=" + fmt.Sprint(p.tokens[0].typ) + ", expected tokenType=" + fmt.Sprint(expected))
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
	for len(p.tokens) > 0 && p.tokens[0].typ == tokenWhitespace {
		p.tokens = p.tokens[1:]
	}
}

func (p *parser) take() *token {
	ret := &p.tokens[0]
	p.tokens = p.tokens[1:]
	return ret
}

func ExtractFields(input string, internalParser FileParser) (map[string]string, error) {
	res, err := internalParser.Extract(input)
	if err != nil {
		return map[string]string{}, err
	}
	return res.Fields, nil
}
