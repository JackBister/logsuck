package config

import "regexp"

type Config struct {
	IndexedFiles []IndexedFileConfig

	// FieldExtractors are regexes. A FieldExtractor should either match one named group where the group name will
	//become the field name and the group content will become the field value,
	//or it should match two groups where the first group will be considered the field name and the second group will be
	//considered the field value.
	// The defaults are [ "(\w+)=(\w+)", "^(?P<_time>\d\d\d\d\/\d\d\/\d\d \d\d:\d\d:\d\d.\d\d\d\d\d\d)"]
	// If a field with the name _time is extracted, it will be matched against TimeLayout
	FieldExtractors []*regexp.Regexp
	// TimeLayout is the layout of the _time field if it is extracted, following Go's time.Parse style https://golang.org/pkg/time/#Parse
	// The default is "2006/01/02 15:04:05"
	TimeLayout string

	EnableWeb bool
	HttpAddr  string
}
