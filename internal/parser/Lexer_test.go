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
	"testing"
)

var tokComma = token{
	typ:   tokenComma,
	value: ",",
}

var tokEquals = token{
	typ:   tokenEquals,
	value: "=",
}

var tokNotEquals = token{
	typ:   tokenNotEquals,
	value: "!=",
}

var tokLparen = token{
	typ:   tokenLparen,
	value: "(",
}

var tokPipe = token{
	typ:   tokenPipe,
	value: "|",
}

var tokRparen = token{
	typ:   tokenRparen,
	value: ")",
}

var tokSpace = token{
	typ:   tokenWhitespace,
	value: " ",
}

func tokKeyword(str string) token {
	return token{
		typ:   tokenKeyword,
		value: str,
	}
}

func tokString(str string) token {
	return token{
		typ:   tokenString,
		value: str,
	}
}

func tokQuoted(str string) token {
	return token{
		typ:   tokenQuotedString,
		value: str,
	}
}

var tokenTests = []struct {
	input           string
	isErrorExpected bool
	expectedTokens  []token
}{
	{
		"", false, []token{},
	},
	{
		" ", false, []token{
			tokSpace,
		},
	},
	{
		"\"quoted\"", false, []token{
			tokQuoted("quoted"),
		},
	},
	{
		"\"quoted with spaces\"", false, []token{
			tokQuoted("quoted with spaces"),
		},
	},
	{
		"\"quoted with \\\"escaped quotes\\\"\"", false, []token{
			tokQuoted("quoted with \"escaped quotes\""),
		},
	},
	{
		"\"multiple\" \"quoted\"\"strings\"", false, []token{
			tokQuoted("multiple"),
			tokSpace,
			tokQuoted("quoted"),
			tokQuoted("strings"),
		},
	},
	{
		"unquoted", false, []token{
			tokString("unquoted"),
		},
	},
	{
		"multiple unquoted strings", false, []token{
			tokString("multiple"),
			tokSpace,
			tokString("unquoted"),
			tokSpace,
			tokString("strings"),
		},
	},
	{
		"\"mixed\" strings", false, []token{
			tokQuoted("mixed"),
			tokSpace,
			tokString("strings"),
		},
	},
	{
		"(", false, []token{
			tokLparen,
		},
	},
	{
		")", false, []token{
			tokRparen,
		},
	},
	{
		"=", false, []token{
			tokEquals,
		},
	},
	{
		"!=", false, []token{
			tokNotEquals,
		},
	},
	{
		"|", false, []token{
			tokPipe,
		},
	},
	{
		",", false, []token{
			tokComma,
		},
	},
	{
		"source=*test* testval IN (\"a bit\", of) | everything", false, []token{
			tokString("source"),
			tokEquals,
			tokString("*test*"),
			tokSpace,
			tokString("testval"),
			tokSpace,
			tokKeyword("IN"),
			tokSpace,
			tokLparen,
			tokQuoted("a bit"),
			tokComma,
			tokSpace,
			tokString("of"),
			tokRparen,
			tokSpace,
			tokPipe,
			tokSpace,
			tokString("everything"),
		},
	},
	{
		"NOT password", false, []token{
			tokKeyword("NOT"),
			tokSpace,
			tokString("password"),
		},
	},
}

func TestLexer_TableTest(t *testing.T) {
	for _, tt := range tokenTests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := tokenize(tt.input)
			if err != nil {
				if tt.isErrorExpected {
					return
				}
				t.Error("Got an error", err)
			}
			if tt.isErrorExpected {
				t.Error("Expected to receive an error")
			}
			if tokens == nil {
				t.Error("Expected non-nil tokens since error was nil")
			}
			if len(tt.expectedTokens) != len(tokens) {
				t.Errorf("Expected %d tokens but got %d", len(tt.expectedTokens), len(tokens))
			}
			for i, tok := range tokens {
				if tt.expectedTokens[i].value != tok.value {
					if len(tt.expectedTokens) < i-1 {
						t.Errorf("Got value '%s' at position %d in the returned token array (out of range of expectedTokens)", tok.value, i)
					} else {
						t.Errorf("Expected value '%s' but got '%s' at position %d in the returned token array", tt.expectedTokens[i].value, tok.value, i)
					}
				}
				if tt.expectedTokens[i].typ != tok.typ {
					if len(tt.expectedTokens) < i-1 {
						t.Errorf("Got type %v at position %d in the returned token array (out of range of expectedTokens)", tok.typ, i)
					} else {
						t.Errorf("Expected type %v but got %v at position %d in the returned token array.", tt.expectedTokens[i].typ, tok.typ, i)
					}
				}
			}
		})
	}
}
