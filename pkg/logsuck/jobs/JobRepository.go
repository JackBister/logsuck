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

type Repository interface {
	AddResults(id int64, events []events.EventIdAndTimestamp) error
	AddTableResults(id int64, tableRows []TableRow) error
	AddFieldStats(id int64, fields []FieldStats) error
	Get(id int64) (*Job, error)
	GetResults(id int64, skip int, take int) (eventIds []int64, err error)
	GetTableResults(id int64, skip int, take int) ([]TableRow, error)
	GetFieldOccurences(id int64) (map[string]int, error)
	GetFieldValues(id int64, fieldName string) (map[string]int, error)
	GetNumMatchedEvents(id int64) (int64, error)
	Insert(query string, startTime, endTime *time.Time, sortMode events.SortMode, outputType pipeline.PipelinePipeType, columnOrder []string) (id *int64, err error)
	UpdateState(id int64, state JobState) error
}

type FieldStats struct {
	Key         string
	Value       string
	Occurrences int
}

type TableRow struct {
	RowNumber int
	Values    map[string]string
}
