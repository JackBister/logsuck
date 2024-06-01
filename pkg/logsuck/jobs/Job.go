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

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type State int32

const (
	StateRunning  State = 1
	StateFinished State = 2
	StateAborted  State = 3
)

type Job struct {
	Id                 int64
	State              State
	Query              string
	StartTime, EndTime *time.Time
	SortMode           events.SortMode
	OutputType         pipeline.PipeType
	ColumnOrder        []string // when OutputType is PipelinePipeTypeTable this is used to decide the order of the columns in the table, otherwise empty
}

type Stats struct {
	EstimatedProgress    float32
	NumMatchedEvents     int64
	FieldOccurences      map[string]int
	FieldValueOccurences map[string]map[string]int
}
