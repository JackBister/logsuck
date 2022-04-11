package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type MapConfigSource struct {
	m       map[string]interface{}
	changes chan struct{}
}

func NewMapConfigSource(m map[string]any) *MapConfigSource {
	return &MapConfigSource{
		m:       m,
		changes: make(chan struct{}),
	}
}

func (mc *MapConfigSource) Changes() <-chan struct{} {
	return mc.changes
}

func (mc *MapConfigSource) Get(name string) (string, bool) {
	split := strings.Split(name, ".")

	current := mc.m
	for i, s := range split {
		c := current[s]
		if m, ok := c.(map[string]interface{}); ok {
			current = m
		} else if i == len(split)-1 {
			rt := reflect.TypeOf(c)
			if ss, ok := c.(string); ok {
				return ss, true
			} else if f, ok := c.(float64); ok {
				return fmt.Sprintf("%g", f), true
			} else if b, ok := c.(bool); ok {
				return strconv.FormatBool(b), true
			} else if a, ok := c.([]interface{}); ok {
				bytes, err := json.Marshal(a)
				if err != nil {
					return "", false
				}
				return string(bytes), true
			} else if rt != nil && rt.Kind() == reflect.Slice {
				bytes, err := json.Marshal(c)
				if err != nil {
					return "", false
				}
				return string(bytes), true
			}
		}
	}
	return "", false
}

func (mc *MapConfigSource) GetKeys() ([]string, bool) {
	return getKeysRecursive(mc.m, "", 0), true
}

func getKeysRecursive(m map[string]interface{}, prefix string, recursionDepth int) []string {
	if recursionDepth > 1000 {
		return []string{}
	}
	ret := make([]string, 0, len(m))
	for k, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			inner := getKeysRecursive(vm, prefix+k+".", recursionDepth+1)
			ret = append(ret, inner...)
		} else {
			ret = append(ret, prefix+k)
		}
	}
	return ret
}
