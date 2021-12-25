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

import (
	"time"

	"github.com/jackbister/logsuck/internal/search"
)

type SortMode = int

const (
	SortModeNone             SortMode = 0
	SortModeTimestampDesc    SortMode = 1
	SortModePreserveArgOrder SortMode = 2
)

type Repository interface {
	AddBatch(events []Event) error
	FilterStream(srch *search.Search, searchStartTime, searchEndTime *time.Time) <-chan []EventWithId
	GetByIds(ids []int64, sortMode SortMode) ([]EventWithId, error)
	GetSurroundingEvents(id int64, count int) ([]EventWithId, error)
}
