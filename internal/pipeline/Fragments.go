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

package pipeline

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
)

func compileMultipleFrags(frags []string) []*regexp.Regexp {
	ret := make([]*regexp.Regexp, 0, len(frags))
	for _, frag := range frags {
		compiled, err := compileFrag(frag)
		if err != nil {
			log.Println("Failed to compile fragment=" + frag + ", err=" + err.Error() + ", fragment will not be included")
		} else {
			ret = append(ret, compiled)
		}
	}
	return ret
}

func compileKeys(m map[string]struct{}) []*regexp.Regexp {
	return compileMultipleFrags(getKeys(m))
}

func getKeys(fragments map[string]struct{}) []string {
	ret := make([]string, 0, len(fragments))
	for k := range fragments {
		ret = append(ret, k)
	}
	return ret
}

func compileFieldValues(m map[string][]string) map[string][]*regexp.Regexp {
	ret := make(map[string][]*regexp.Regexp, len(m))
	for key, values := range m {
		compiledValues := make([]*regexp.Regexp, len(values))
		for i, value := range values {
			compiled, err := compileFrag(value)
			if err != nil {
				log.Println("Failed to compile fieldValue=" + value + ", err=" + err.Error() + ", fieldValue will not be included")
			} else {
				compiledValues[i] = compiled
			}
		}
		ret[key] = compiledValues
	}
	return ret
}

func compileFrag(frag string) (*regexp.Regexp, error) {
	pre := "(^|\\W)"
	if strings.HasPrefix(frag, "*") {
		pre = ""
	}
	post := "($|\\W)"
	if strings.HasSuffix(frag, "*") {
		post = ""
	}
	rexString := pre + strings.Replace(frag, "*", ".*", -1) + post
	rex, err := regexp.Compile(rexString)
	if err != nil {
		return nil, fmt.Errorf("Failed to compile rexString="+rexString+": %w", err)
	}
	return rex, nil
}

func shouldIncludeEvent(evt events.EventWithId,
	cfg *config.StaticConfig,
	compiledFrags []*regexp.Regexp, compiledNotFrags []*regexp.Regexp,
	compiledFields map[string][]*regexp.Regexp, compiledNotFields map[string][]*regexp.Regexp) (map[string]string, bool) {
	evtFields := parser.ExtractFields(strings.ToLower(evt.Raw), cfg.FieldExtractors)
	// TODO: This could produce unexpected results
	evtFields["host"] = evt.Host
	evtFields["source"] = evt.Source

	include := true
	for key, values := range compiledFields {
		evtValue, ok := evtFields[key]
		if !ok {
			include = false
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if !anyMatch {
			include = false
			break
		}
	}
	for key, values := range compiledNotFields {
		evtValue, ok := evtFields[key]
		if !ok {
			break
		}
		anyMatch := false
		for _, value := range values {
			if value.MatchString(evtValue) {
				anyMatch = true
			}
		}
		if anyMatch {
			include = false
			break
		}
	}
	return evtFields, include
}
