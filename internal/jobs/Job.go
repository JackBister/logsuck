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

package jobs

import (
	"time"

	"github.com/jackbister/logsuck/internal/events"
)

type JobState int32

const (
	JobStateRunning  JobState = 1
	JobStateFinished JobState = 2
	JobStateAborted  JobState = 3
)

type Job struct {
	Id                 int64
	State              JobState
	Query              string
	StartTime, EndTime *time.Time
	SortMode           events.SortMode
}

type JobStats struct {
	EstimatedProgress    float32
	NumMatchedEvents     int64
	FieldOccurences      map[string]int
	FieldValueOccurences map[string]map[string]int
}
