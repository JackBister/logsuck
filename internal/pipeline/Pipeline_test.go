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

package pipeline

import (
	"reflect"
	"testing"

	"github.com/jackbister/logsuck/internal/events"
)

func TestIgnoresPreviousStepsOptimization(t *testing.T) {
	p, err := CompilePipeline("| search \"abc\" | search \"def\"", nil, nil)
	if err != nil {
		t.Fatalf("got error when compiling pipeline: %v", err)
		return
	}
	if len(p.steps) != 1 {
		t.Fatalf("unexpected number of steps in pipeline, expected=%v, got=%v", 1, len(p.steps))
		return
	}
	sps, ok := p.steps[0].(*searchPipelineStep)
	if !ok {
		t.Fatalf("unexpected step type for step 0 in pipeline, expected=searchPipelineStep, got=%v", reflect.TypeOf(p.steps[0]).Name())
		return
	}
	_, ok = sps.srch.Fragments["def"]
	if !ok {
		t.Fatalf("expected step 0 to contain the fragment \"def\", have=%v", sps.srch.Fragments)
		return
	}
}

func TestColumnOrder_OutputTypeNotTable(t *testing.T) {
	p, _ := CompilePipeline("", nil, nil)
	columnOrder, err := p.ColumnOrder()
	if err != nil {
		t.Error("got error when getting column order for 'everything' pipeline", err)
	}
	if columnOrder == nil {
		t.Error("got nil column order for 'everything' pipeline")
	}
	if len(columnOrder) != 0 {
		t.Error("got unexpected columnOrder for 'everything' pipeline. Since this pipeline does not generate a table it should have an empty columnOrder", columnOrder)
	}
}

func TestColumnOrder_OutputTypeTable(t *testing.T) {
	p, _ := CompilePipeline("| table \"host, source, _time\"", nil, nil)
	columnOrder, err := p.ColumnOrder()
	if err != nil {
		t.Error("got error when getting column order for pipeline with table step", err)
	}
	if columnOrder[0] != "host" {
		t.Errorf("unexpected columnOrder, expected \"host\" at index 0 but have %v", columnOrder[0])
	}
	if columnOrder[1] != "source" {
		t.Errorf("unexpected columnOrder, expected \"source\" at index 1 but have %v", columnOrder[1])
	}
	if columnOrder[2] != "_time" {
		t.Errorf("unexpected columnOrder, expected \"_time\" at index 0 but have %v", columnOrder[2])
	}
}

func TestSortMode_EverythingPipeline(t *testing.T) {
	p, _ := CompilePipeline("", nil, nil)
	if p.SortMode() != events.SortModeTimestampDesc {
		t.Error("unexpected sortMode, expected SortModeTimestampDesc for 'everything' pipeline", p.SortMode())
	}
}

func TestSortMode_SurroundingPipeline(t *testing.T) {
	p, _ := CompilePipeline("| surrounding eventId=1", nil, nil)
	if p.SortMode() != events.SortModePreserveArgOrder {
		t.Error("unexpected sortMode, expected SortModePreserveArgOrder for surrounding pipeline", p.SortMode())
	}
}

func TestTypePropagation_Events(t *testing.T) {
	p, _ := CompilePipeline("| where x=y", nil, nil)
	if p.OutputType() != PipelinePipeTypeEvents {
		t.Error("unexpected output type, expected PipelinePipeTypeEvents since the where pipe should propagate the search pipe's output type", p.OutputType())
	}
}

func TestTypePropagation_Table(t *testing.T) {
	p, _ := CompilePipeline("| table \"x\" | where x=y", nil, nil)
	if p.OutputType() != PipelinePipeTypeTable {
		t.Error("unexpected output type, expected PipelinePipeTypeTable since the where pipe should propagate the table pipe's output type", p.OutputType())
	}
}
