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
