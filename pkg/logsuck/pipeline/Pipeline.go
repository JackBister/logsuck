package pipeline

import (
	"context"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
)

type PipelinePipeType int

const (
	PipelinePipeTypeNone   PipelinePipeType = 0
	PipelinePipeTypeEvents PipelinePipeType = 1
	PipelinePipeTypeTable  PipelinePipeType = 2

	PipelinePipeTypePropagate PipelinePipeType = 999
)

type PipelineParameters struct {
	ConfigSource config.ConfigSource
	EventsRepo   events.Repository

	Logger *slog.Logger
}

type PipelineStepResult struct {
	Events    []events.EventWithExtractedFields
	TableRows []map[string]string
}

type PipelinePipe struct {
	Input      <-chan PipelineStepResult
	InputType  PipelinePipeType
	Output     chan<- PipelineStepResult
	OutputType PipelinePipeType
}

type PipelineStep interface {
	Execute(ctx context.Context, pipe PipelinePipe, params PipelineParameters)

	// Returns the name of the operator that created this step, for example "rex"
	Name() string

	InputType() PipelinePipeType
	OutputType() PipelinePipeType
}

type PipelineStepWithSortMode interface {
	SortMode() events.SortMode
}

type TableGeneratingPipelineStep interface {
	ColumnOrder() []string
}

type StepCompiler func(input string, options map[string]string) (PipelineStep, error)

type StepDefinition struct {
	StepName string
	Compiler StepCompiler
}
