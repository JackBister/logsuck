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

type StaticConfig struct {
	// FieldExtractors are regexes. A FieldExtractor should either match one named group where the group name will
	//become the field name and the group content will become the field value,
	//or it should match two groups where the first group will be considered the field name and the second group will be
	//considered the field value.
	// The defaults are [ "(\w+)=(\w+)", "^(?P<_time>\d\d\d\d\/\d\d\/\d\d \d\d:\d\d:\d\d.\d\d\d\d\d\d)"]
	// If a field with the name _time is extracted, it will be matched against TimeLayout
	FieldExtractors []*regexp.Regexp

	HostName string

	ConfigPollInterval time.Duration

	Forwarder *ForwarderConfig
	Recipient *RecipientConfig

	SQLite *SqliteConfig

	Tasks *TasksConfig

	Web *WebConfig
}
