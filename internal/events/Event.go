// Copyright 2020 The Logsuck Authors
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

package events

import "time"

// RawEvent represents an Event that has not yet been enriched with information about field values etc.
type RawEvent struct {
	Raw    string
	Host   string
	Source string
	Offset int64
}

type Event struct {
	Raw       string
	Timestamp time.Time
	Host      string
	Source    string
	Offset    int64
}

type EventWithId struct {
	Id        int64
	Raw       string
	Timestamp time.Time
	Host      string
	Source    string
}

type EventWithExtractedFields struct {
	Id        int64
	Raw       string
	Timestamp time.Time
	Source    string
	Fields    map[string]string
}

type EventIdAndTimestamp struct {
	Id        int64
	Timestamp time.Time
}
