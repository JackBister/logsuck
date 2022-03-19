package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJsonConfigSource_BasicCase(t *testing.T) {
	jc := &MapConfigSource{
		m: map[string]interface{}{
			"a": "my string",
			"b": 1.23,
			"c": true,
		},
	}
	assertValueEquals(jc, "a", "my string", t)
	assertValueEquals(jc, "b", "1.23", t)
	assertValueEquals(jc, "c", "true", t)
}

func TestJsonConfigSource_NestedField(t *testing.T) {
	jc := createJsonConfigSource(`{ "a": { "b": "my string" } }`, t)
	assertValueEquals(jc, "a.b", "my string", t)
}

func TestJsonConfigSource_ArrayValue(t *testing.T) {
	jc := createJsonConfigSource(`{ "a": [ 1, 2, 3 ] }`, t)
	assertValueEquals(jc, "a", "[1,2,3]", t)
}

func createJsonConfigSource(s string, t *testing.T) *MapConfigSource {
	decoder := json.NewDecoder(strings.NewReader(s))
	m := map[string]interface{}{}
	err := decoder.Decode(&m)
	if err != nil {
		t.Fatalf("got error when decoding json: %v", err)
	}
	jc := &MapConfigSource{
		m: m,
	}
	return jc
}

func assertValueEquals(jc *MapConfigSource, key string, expected string, t *testing.T) {
	val, ok := jc.Get(key)
	if !ok {
		t.Fatalf("!ok when getting '%s' from JsonConfigSource", key)
	}
	if val != expected {
		t.Fatalf("expected key '%s' to return value '%s' but got '%s'", key, expected, val)
	}
}
