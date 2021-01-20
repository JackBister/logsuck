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
	"log"
	"regexp"
)

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
	if len(p.tokens) == 0 {
		return
	}
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
