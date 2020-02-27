package parser

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

type tokenType int

const (
	tokenString       tokenType = 0
	tokenQuotedString           = 1
	tokenWhitespace             = 2
	tokenEquals                 = 3
	tokenLparen                 = 4
	tokenRparen                 = 5
	tokenPipe                   = 6
	tokenComma                  = 7
	tokenKeyword                = 8

	tokenInvalid = 0xBEEF
)

type token struct {
	typ   tokenType
	value string
}

type tokenizer struct {
	tokens        []token
	insideString  bool
	currentString strings.Builder
}

var keywords = [...]string{
	"in",
	"not",
}

const symbols = "=|(),"
const whiteSpace = " \n\t"

var wordDelimiters = symbols + whiteSpace

func tokenize(input string) ([]token, error) {
	tk := tokenizer{
		tokens:        make([]token, 0, 1),
		insideString:  false,
		currentString: strings.Builder{},
	}
	return tk.tokenize(input)
}

func (tk *tokenizer) tokenize(input string) ([]token, error) {
	stringEndRegexp := regexp.MustCompile("[^\\\\]\"")

	for i := 0; i < len(input); i++ {
		r, _ := utf8.DecodeRuneInString(input[i:])
		if strings.ContainsRune(whiteSpace, r) {
			tk.addToken(token{
				typ:   tokenWhitespace,
				value: string(r),
			})
		} else if r == '=' {
			tk.addToken(token{
				typ:   tokenEquals,
				value: "=",
			})
		} else if r == '(' {
			tk.addToken(token{
				typ:   tokenLparen,
				value: "(",
			})
		} else if r == ')' {
			tk.addToken(token{
				typ:   tokenRparen,
				value: ")",
			})
		} else if r == '|' {
			tk.addToken(token{
				typ:   tokenPipe,
				value: "|",
			})
		} else if r == ',' {
			tk.addToken(token{
				typ:   tokenComma,
				value: ",",
			})
		} else if r == '"' {
			if i == len(input)-1 {
				return nil, errors.New("unclosed quote at end of string")
			}
			remainder := input[i+1:]
			endLocation := stringEndRegexp.FindStringIndex(remainder)
			if len(endLocation) == 0 {
				return nil, errors.New("Unclosed quote at offset " + string(i))
			}
			str := remainder[:endLocation[0]+1]
			str = strings.ReplaceAll(str, "\\\"", "\"")
			tk.addToken(token{
				typ:   tokenQuotedString,
				value: str,
			})
			i += endLocation[1]
		} else {
			remainder := input[i:]
			endLocation := strings.IndexAny(remainder, wordDelimiters)
			var str string
			if endLocation == -1 {
				str = remainder
			} else {
				str = remainder[:endLocation]
			}

			strLowered := strings.ToLower(str)
			isKeyword := false
			for _, kw := range keywords {
				if kw == strLowered {
					isKeyword = true
				}
			}

			if isKeyword {
				tk.addToken(token{
					typ:   tokenKeyword,
					value: strLowered,
				})
			} else {
				tk.addToken(token{
					typ:   tokenString,
					value: str,
				})
			}

			if endLocation == -1 {
				break
			}
			i += endLocation - 1
		}
	}

	return tk.tokens, nil
}

func (tk *tokenizer) handleQuote(str string, quoteIndex int) {
	if quoteIndex == 0 || str[quoteIndex-1] != '\\' {
		if tk.insideString {
			tk.currentString.WriteString(str[:quoteIndex])
			tk.addToken(token{
				typ:   tokenString,
				value: tk.currentString.String(),
			})
			tk.currentString.Reset()
			tk.insideString = false
		} else {
			tk.currentString.WriteString(str[quoteIndex:])
			tk.insideString = true
		}
	}
}

func (tk *tokenizer) addToken(t token) {
	tk.tokens = append(tk.tokens, t)
}
