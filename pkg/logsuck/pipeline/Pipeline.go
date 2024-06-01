// Copyright 2024 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipeline

import (
	"context"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
)

type PipeType int

const (
	PipeTypeNone   PipeType = 0
	PipeTypeEvents PipeType = 1
	PipeTypeTable  PipeType = 2

	PipeTypePropagate PipeType = 999
)

type Parameters struct {
	ConfigSource config.Source
	EventsRepo   events.Repository

	Logger *slog.Logger
}

type StepResult struct {
	Events    []events.EventWithExtractedFields
	TableRows []map[string]string
}

type Pipe struct {
	Input      <-chan StepResult
	InputType  PipeType
	Output     chan<- StepResult
	OutputType PipeType
}

type Step interface {
	Execute(ctx context.Context, pipe Pipe, params Parameters)

	// Returns the name of the operator that created this step, for example "rex"
	Name() string

	InputType() PipeType
	OutputType() PipeType
}

type StepWithSortMode interface {
	SortMode() events.SortMode
}

type TableGeneratingStep interface {
	ColumnOrder() []string
}

type StepCompiler func(input string, options map[string]string) (Step, error)

type StepDefinition struct {
	StepName string
	Compiler StepCompiler
}
