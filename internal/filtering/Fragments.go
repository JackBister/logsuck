package filtering

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

func CompileMultiple(frags []string) []*regexp.Regexp {
	ret := make([]*regexp.Regexp, 0, len(frags))
	for _, frag := range frags {
		compiled, err := Compile(frag)
		if err != nil {
			log.Println("Failed to compile fragment=" + frag + ", err=" + err.Error() + ", fragment will not be included")
		} else {
			ret = append(ret, compiled)
		}
	}
	return ret
}

func CompileKeys(m map[string]struct{}) []*regexp.Regexp {
	return CompileMultiple(getKeys(m))
}

func getKeys(fragments map[string]struct{}) []string {
	ret := make([]string, 0, len(fragments))
	for k := range fragments {
		ret = append(ret, k)
	}
	return ret
}

func CompileMap(m map[string][]string) map[string][]*regexp.Regexp {
	ret := make(map[string][]*regexp.Regexp, len(m))
	for key, values := range m {
		compiledValues := make([]*regexp.Regexp, len(values))
		for i, value := range values {
			compiled, err := Compile(value)
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

func Compile(frag string) (*regexp.Regexp, error) {
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
