package events

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jackbister/logsuck/internal/search"
)

type Repository interface {
	Add(evt Event)
	Search(srch *search.Search) []Event
}

type inMemoryRepository struct {
	events []Event
}

func InMemoryRepository() Repository {
	return &inMemoryRepository{
		events: make([]Event, 0),
	}
}

func (repo *inMemoryRepository) Add(evt Event) {
	// TODO: thread safety
	repo.events = append(repo.events, evt)
}

func (repo *inMemoryRepository) Search(srch *search.Search) []Event {
	ret := make([]Event, 0, 1)
	compiledFrags := compileFrags(getKeys(srch.Fragments))
	compiledNotFrags := compileFrags(getKeys(srch.NotFragments))
	compiledFields := compileFields(srch.Fields)
	compiledNotFields := compileFields(srch.NotFields)
	for _, evt := range repo.events {
		rawLowered := strings.ToLower(evt.Raw)

		include := true
		for _, frag := range compiledFrags {
			if !frag.MatchString(rawLowered) {
				include = false
				break
			}
		}
		for _, frag := range compiledNotFrags {
			if frag.MatchString(rawLowered) {
				include = false
				break
			}
		}
		for key, values := range compiledFields {
			evtValue, ok := evt.Fields[key]
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
			evtValue, ok := evt.Fields[key]
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

		if include {
			ret = append(ret, evt)
		}
	}
	return ret
}

func compileFields(fields map[string][]string) map[string][]*regexp.Regexp {
	ret := make(map[string][]*regexp.Regexp, len(fields))
	for key, values := range fields {
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

func compileFrags(frags []string) []*regexp.Regexp {
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

func getKeys(fragments map[string]struct{}) []string {
	ret := make([]string, 0, len(fragments))
	for k := range fragments {
		ret = append(ret, k)
	}
	return ret
}
