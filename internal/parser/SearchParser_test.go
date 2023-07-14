package parser

import "testing"

var tableTests = []struct {
	input                string
	expectedFragments    []string
	expectedNotFragments []string
	expectedFields       map[string][]string
	expectedNotFields    map[string][]string
}{
	{
		"msg",
		[]string{"msg"},
		[]string{},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"\"msg\"",
		[]string{"msg"},
		[]string{},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"NOT msg",
		[]string{},
		[]string{"msg"},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"NOT \"msg\"",
		[]string{},
		[]string{"msg"},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"msg NOT msg2",
		[]string{"msg"},
		[]string{"msg2"},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"msg NOT \"msg2\"",
		[]string{"msg"},
		[]string{"msg2"},
		map[string][]string{},
		map[string][]string{},
	},
	{
		"msg=msg2",
		[]string{},
		[]string{},
		map[string][]string{
			"msg": {"msg2"},
		},
		map[string][]string{},
	},
	{
		"msg=\"msg2\"",
		[]string{},
		[]string{},
		map[string][]string{
			"msg": {"msg2"},
		},
		map[string][]string{},
	},
	{
		"msg=msg2 msg=msg3",
		[]string{},
		[]string{},
		map[string][]string{
			"msg": {"msg3"},
		},
		map[string][]string{},
	},
	{
		"msg IN (msg2, msg3)",
		[]string{},
		[]string{},
		map[string][]string{
			"msg": {"msg2", "msg3"},
		},
		map[string][]string{},
	},
	{
		"msg NOT IN (msg2, msg3)",
		[]string{},
		[]string{},
		map[string][]string{},
		map[string][]string{
			"msg": {"msg2", "msg3"},
		},
	},
}

func TestSearchParser_TableTest(t *testing.T) {
	for _, tt := range tableTests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := ParseSearch(tt.input)
			if err != nil {
				t.Error("got error when parsing input", err)
			}
			checkFragments(t, tt.expectedFragments, res.Fragments, "Fragments")
			checkFragments(t, tt.expectedNotFragments, res.NotFragments, "NotFragments")
			checkFields(t, tt.expectedFields, res.Fields, "Fields")
			checkFields(t, tt.expectedNotFields, res.NotFields, "NotFields")
		})
	}
}

func checkFragments(t *testing.T, expectedFragments []string, actualFragments map[string]struct{}, name string) {
	if len(actualFragments) != len(expectedFragments) {
		t.Errorf("%v: got unexpected number of fragments. expected=%v, actual=%v", name, len(expectedFragments), len(actualFragments))
	}
	for _, f := range expectedFragments {
		if _, ok := actualFragments[f]; !ok {
			t.Errorf("%v: did not get expected fragment=%v", name, f)
		}
	}
}

func checkFields(t *testing.T, expectedFields map[string][]string, actualFields map[string][]string, name string) {
	if len(actualFields) != len(expectedFields) {
		t.Errorf("%v: got unexpected number of fields. expected=%v, actual=%v", name, len(expectedFields), len(actualFields))
	}
	for k, v := range expectedFields {
		if v2, ok := actualFields[k]; !ok {
			t.Errorf("%v: did not get expected field=%v", name, k)
		} else {
			if len(v) != len(v2) {
				t.Errorf("%v: got unexpected number of values for field=%v. expected=%v, actual=%v", name, k, len(v), len(v2))
			}
			for _, fv := range v {
				ok := false
				for _, fv2 := range v2 {
					if fv2 == fv {
						ok = true
						break
					}
				}
				if !ok {
					t.Errorf("%v: did not find expected value=%v for field=%v", name, fv, k)
				}
			}
		}
	}
}
