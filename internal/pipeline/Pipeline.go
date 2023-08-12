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
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
)

type PipelinePipeType int

const (
	PipelinePipeTypeNone   PipelinePipeType = 0
	PipelinePipeTypeEvents PipelinePipeType = 1
	PipelinePipeTypeTable  PipelinePipeType = 2

	PipelinePipeTypePropagate PipelinePipeType = 999
)

type Pipeline struct {
	steps   []pipelineStep
	pipes   []pipelinePipe
	outChan <-chan PipelineStepResult
}

type PipelineParameters struct {
	ConfigSource config.ConfigSource
	EventsRepo   events.Repository

	Logger *slog.Logger
}

type PipelineStepResult struct {
	Events    []events.EventWithExtractedFields
	TableRows []map[string]string
}

// TODO: What is a reasonable value? Configurable? Dynamic?
// pipeBufferSize determines the buffer size of the channels between the steps in a pipeline.
// A low value means a step later in the pipe can cause an earlier step to slow down as it needs to wait for the rest
// of the pipe to catch up, while a high value can lead to high memory consumption and probably some other issues
// A buffer size of 0 means the entire pipeline must process each batch before a new batch can added to the start of
// the pipe.
const pipeBufferSize = 100

type pipelinePipe struct {
	input      <-chan PipelineStepResult
	inputType  PipelinePipeType
	output     chan<- PipelineStepResult
	outputType PipelinePipeType
}

type pipelineStep interface {
	Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters)

	// Returns the name of the operator that created this step, for example "rex"
	Name() string

	InputType() PipelinePipeType
	OutputType() PipelinePipeType
}

type pipelineStepWithSortMode interface {
	SortMode() events.SortMode
}

type tableGeneratingPipelineStep interface {
	ColumnOrder() []string
}

var compilers = map[string]func(input string, options map[string]string) (pipelineStep, error){
	"rex":         compileRexStep,
	"search":      compileSearchStep,
	"surrounding": compileSurroundingStep,
	"table":       compileTableStep,
	"where":       compileWhereStep,
}

func CompilePipeline(input string, startTime, endTime *time.Time) (*Pipeline, error) {
	pr, err := parser.ParsePipeline(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile pipeline: %w", err)
	}

	compiledSteps := make([]pipelineStep, len(pr.Steps))
	for i, step := range pr.Steps {
		compiler, ok := compilers[step.StepType]
		if !ok {
			return nil, fmt.Errorf("failed to compile pipeline: no compiler found for StepType=%v", step.StepType)
		}
		// This feels pretty dumb
		if i == 0 && step.StepType == "search" {
			if startTime != nil {
				step.Args["startTime"] = startTime.Format(time.RFC3339Nano)
			}
			if endTime != nil {
				step.Args["endTime"] = endTime.Format(time.RFC3339Nano)
			}
		}
		res, err := compiler(step.Value, step.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to compile pipeline: failed to compile step %v: %w", i, err)
		}
		compiledSteps[i] = res
	}

	lastGeneratorIndex := 0
	for i, compiled := range compiledSteps {
		if compiled.InputType() == PipelinePipeTypeNone {
			lastGeneratorIndex = i
		}
	}
	compiledSteps = compiledSteps[lastGeneratorIndex:]

	outputType := compiledSteps[0].OutputType()
	for i, compiled := range compiledSteps {
		if (compiled.InputType() == PipelinePipeTypePropagate || compiled.OutputType() == PipelinePipeTypePropagate) && compiled.InputType() != compiled.OutputType() {
			return nil, fmt.Errorf("failed to compile pipeline: mismatching input/output type for propagating step. input=%v, output=%v. This is a bug", compiled.InputType(), compiled.OutputType())
		}
		if compiled.OutputType() != PipelinePipeTypePropagate {
			outputType = compiled.OutputType()
		}
		if i == len(compiledSteps)-1 {
			if outputType != PipelinePipeTypeEvents && outputType != PipelinePipeTypeTable {
				return nil, fmt.Errorf("failed to compile pipeline: invalid output type for last step: %v", compiled.Name())
			}
		} else {
			if outputType != compiledSteps[i+1].InputType() && compiledSteps[i+1].InputType() != PipelinePipeTypePropagate {
				return nil, fmt.Errorf("failed to compile pipeline: output type for step %v does not match input type for step %v", compiled.Name(), compiledSteps[i+1].Name())
			}
		}
	}

	lastOutput := make(chan PipelineStepResult, pipeBufferSize)
	lastOutputType := compiledSteps[0].OutputType()
	close(lastOutput)
	pipes := make([]pipelinePipe, len(compiledSteps))
	for i := 0; i < len(compiledSteps); i++ {
		currentOutputType := compiledSteps[i].OutputType()
		if currentOutputType == PipelinePipeTypePropagate {
			currentOutputType = lastOutputType
		}
		outputEvents := make(chan PipelineStepResult, pipeBufferSize)
		pipes[i] = pipelinePipe{
			input:      lastOutput,
			inputType:  lastOutputType,
			output:     outputEvents,
			outputType: currentOutputType,
		}
		lastOutput = outputEvents
		lastOutputType = currentOutputType
	}

	return &Pipeline{
		steps:   compiledSteps,
		pipes:   pipes,
		outChan: lastOutput,
	}, nil
}

func (p *Pipeline) ColumnOrder() ([]string, error) {
	lastStep := p.steps[len(p.steps)-1]
	if lastStep.OutputType() != PipelinePipeTypeTable {
		return []string{}, nil
	}
	if t, ok := lastStep.(tableGeneratingPipelineStep); !ok {
		return []string{}, fmt.Errorf("failed to cast step=%v to tableGeneratingPipelineStep despite OutputType being PipelinePipeTypeTable. This is likely a bug! stepName=%v",
			lastStep, lastStep.Name())
	} else {
		return t.ColumnOrder(), nil
	}
}

func (p *Pipeline) Execute(ctx context.Context, params PipelineParameters) <-chan PipelineStepResult {
	for i, step := range p.steps {
		go step.Execute(ctx, p.pipes[i], params)
	}
	return p.outChan
}

func (p *Pipeline) GetStepNames() []string {
	ret := make([]string, len(p.steps))
	for i, s := range p.steps {
		ret[i] = s.Name()
	}
	return ret
}

func (p *Pipeline) OutputType() PipelinePipeType {
	outputType := p.steps[0].OutputType()
	for _, s := range p.steps {
		if s.OutputType() != PipelinePipeTypePropagate {
			outputType = s.OutputType()
		}
	}
	return outputType
}

func (p *Pipeline) SortMode() events.SortMode {
	sortMode := events.SortModeTimestampDesc
	for _, s := range p.steps {
		if ss, ok := s.(pipelineStepWithSortMode); ok {
			sortMode = ss.SortMode()
		}
	}
	return sortMode
}
