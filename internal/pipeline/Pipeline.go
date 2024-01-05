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
	"time"

	"github.com/jackbister/logsuck/internal/parser"
	"go.uber.org/dig"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	api "github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type Pipeline struct {
	steps   []api.Step
	pipes   []api.Pipe
	outChan <-chan api.StepResult
}

// TODO: What is a reasonable value? Configurable? Dynamic?
// pipeBufferSize determines the buffer size of the channels between the steps in a pipeline.
// A low value means a step later in the pipe can cause an earlier step to slow down as it needs to wait for the rest
// of the pipe to catch up, while a high value can lead to high memory consumption and probably some other issues
// A buffer size of 0 means the entire pipeline must process each batch before a new batch can added to the start of
// the pipe.
const pipeBufferSize = 100

type PipelineCompiler struct {
	stepDefinitions map[string]api.StepDefinition
}

func NewPipelineCompiler(p struct {
	dig.In

	StepDefinitions []api.StepDefinition `group:"steps"`
}) PipelineCompiler {
	m := map[string]api.StepDefinition{}
	for _, v := range p.StepDefinitions {
		m[v.StepName] = v
	}
	return PipelineCompiler{
		stepDefinitions: m,
	}
}

func (pc *PipelineCompiler) Compile(input string, startTime, endTime *time.Time) (*Pipeline, error) {
	pr, err := parser.ParsePipeline(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile pipeline: %w", err)
	}

	compiledSteps := make([]api.Step, len(pr.Steps))
	for i, step := range pr.Steps {
		stepDefinition, ok := pc.stepDefinitions[step.StepType]
		if !ok {
			return nil, fmt.Errorf("failed to compile pipeline: no step definition found for StepType=%v", step.StepType)
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
		res, err := stepDefinition.Compiler(step.Value, step.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to compile pipeline: failed to compile step %v: %w", i, err)
		}
		compiledSteps[i] = res
	}

	lastGeneratorIndex := 0
	for i, compiled := range compiledSteps {
		if compiled.InputType() == api.PipeTypeNone {
			lastGeneratorIndex = i
		}
	}
	compiledSteps = compiledSteps[lastGeneratorIndex:]

	outputType := compiledSteps[0].OutputType()
	for i, compiled := range compiledSteps {
		if (compiled.InputType() == api.PipeTypePropagate || compiled.OutputType() == api.PipeTypePropagate) && compiled.InputType() != compiled.OutputType() {
			return nil, fmt.Errorf("failed to compile pipeline: mismatching input/output type for propagating step. input=%v, output=%v. This is a bug", compiled.InputType(), compiled.OutputType())
		}
		if compiled.OutputType() != api.PipeTypePropagate {
			outputType = compiled.OutputType()
		}
		if i == len(compiledSteps)-1 {
			if outputType != api.PipeTypeEvents && outputType != api.PipeTypeTable {
				return nil, fmt.Errorf("failed to compile pipeline: invalid output type for last step: %v", compiled.Name())
			}
		} else {
			if outputType != compiledSteps[i+1].InputType() && compiledSteps[i+1].InputType() != api.PipeTypePropagate {
				return nil, fmt.Errorf("failed to compile pipeline: output type for step %v does not match input type for step %v", compiled.Name(), compiledSteps[i+1].Name())
			}
		}
	}

	lastOutput := make(chan api.StepResult, pipeBufferSize)
	lastOutputType := compiledSteps[0].OutputType()
	close(lastOutput)
	pipes := make([]api.Pipe, len(compiledSteps))
	for i := 0; i < len(compiledSteps); i++ {
		currentOutputType := compiledSteps[i].OutputType()
		if currentOutputType == api.PipeTypePropagate {
			currentOutputType = lastOutputType
		}
		outputEvents := make(chan api.StepResult, pipeBufferSize)
		pipes[i] = api.Pipe{
			Input:      lastOutput,
			InputType:  lastOutputType,
			Output:     outputEvents,
			OutputType: currentOutputType,
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
	if lastStep.OutputType() != api.PipeTypeTable {
		return []string{}, nil
	}
	if t, ok := lastStep.(api.TableGeneratingStep); !ok {
		return []string{}, fmt.Errorf("failed to cast step=%v to tableGeneratingPipelineStep despite OutputType being PipelinePipeTypeTable. This is likely a bug! stepName=%v",
			lastStep, lastStep.Name())
	} else {
		return t.ColumnOrder(), nil
	}
}

func (p *Pipeline) Execute(ctx context.Context, params api.Parameters) <-chan api.StepResult {
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

func (p *Pipeline) OutputType() api.PipeType {
	outputType := p.steps[0].OutputType()
	for _, s := range p.steps {
		if s.OutputType() != api.PipeTypePropagate {
			outputType = s.OutputType()
		}
	}
	return outputType
}

func (p *Pipeline) SortMode() events.SortMode {
	sortMode := events.SortModeTimestampDesc
	for _, s := range p.steps {
		if ss, ok := s.(api.StepWithSortMode); ok {
			sortMode = ss.SortMode()
		}
	}
	return sortMode
}
