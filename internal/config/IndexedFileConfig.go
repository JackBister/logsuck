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

package config

import (
	"regexp"
	"time"
)

// IndexedFileConfig contains configuration for a specific file which will be indexed
type IndexedFileConfig struct {
	// Filename is the name of the file. It can also be a glob pattern. If the glob pattern matches multiple files, multiple watchers will be started.
	Filename string
	// EventDelimiter is a regex that is used to determine where one event ends and another begins.
	// The default is "\n".
	EventDelimiter *regexp.Regexp
	// ReadInterval is the time the file watcher will sleep between looking for new events in the file.
	// A lower duration will make events arrive faster in the search engine, but will consume more CPU.
	// The default is 10 * time.Second.
	ReadInterval time.Duration
	// TimeLayout is the layout of the _time field if it is extracted, following Go's time.Parse style https://golang.org/pkg/time/#Parse
	// The default is "2006/01/02 15:04:05"
	TimeLayout string
}

func FileConfigFromDynamicArray(a []interface{}) []IndexedFileConfig {
	ret := make([]IndexedFileConfig, 0, len(a))
	for _, o := range a {
		if m, ok := o.(map[string]interface{}); !ok {
			continue
		} else {
			filename, ok := m["fileName"].(string)
			if !ok {
				continue
			}
			eventDelimiter, ok := m["eventDelimiter"].(string)
			if !ok {
				eventDelimiter = "\n"
			}
			eventDelimiterRegex, err := regexp.Compile(eventDelimiter)
			if err != nil {
				continue
			}
			readInterval, ok := m["readInterval"].(string)
			if !ok {
				readInterval = "1s"
			}
			readIntervalDuration, err := time.ParseDuration(readInterval)
			if err != nil {
				continue
			}
			timeLayout, ok := m["timeLayout"].(string)
			if !ok {
				timeLayout = "2006/01/02 15:04:05"
			}
			ret = append(ret, IndexedFileConfig{
				Filename:       filename,
				EventDelimiter: eventDelimiterRegex,
				ReadInterval:   readIntervalDuration,
				TimeLayout:     timeLayout,
			})
		}
	}
	return ret
}
