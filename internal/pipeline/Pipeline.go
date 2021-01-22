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

package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
)

type Pipeline struct {
	steps   []pipelineStep
	pipes   []pipelinePipe
	outChan <-chan PipelineStepResult
}

type PipelineParameters struct {
	Cfg        *config.Config
	EventsRepo events.Repository
}

type PipelineStepResult struct {
	Events []events.EventWithExtractedFields
}

// TODO: What is a reasonable value? Configurable? Dynamic?
// pipeBufferSize determines the buffer size of the channels between the steps in a pipeline.
// A low value means a step later in the pipe can cause an earlier step to slow down as it needs to wait for the rest
// of the pipe to catch up, while a high value can lead to high memory consumption and probably some other issues
// A buffer size of 0 means the entire pipeline must process each batch before a new batch can added to the start of
// the pipe.
const pipeBufferSize = 100

type pipelinePipe struct {
	input  <-chan PipelineStepResult
	output chan<- PipelineStepResult
}

type pipelineStep interface {
	Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters)
}

var compilers = map[string]func(input string, options map[string]string) (pipelineStep, error){
	"rex":    compileRexStep,
	"search": compileSearchStep,
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

	lastOutput := make(chan PipelineStepResult, pipeBufferSize)
	close(lastOutput)
	pipes := make([]pipelinePipe, len(compiledSteps))
	for i := 0; i < len(pr.Steps); i++ {
		outputEvents := make(chan PipelineStepResult, pipeBufferSize)
		pipes[i] = pipelinePipe{
			input:  lastOutput,
			output: outputEvents,
		}
		lastOutput = outputEvents
	}

	log.Println("outchan", lastOutput)
	return &Pipeline{
		steps:   compiledSteps,
		pipes:   pipes,
		outChan: lastOutput,
	}, nil
}

func (p *Pipeline) Execute(ctx context.Context, params PipelineParameters) <-chan PipelineStepResult {
	for i, step := range p.steps {
		log.Printf("pipe %v %v", i, p.pipes[i])
		go step.Execute(ctx, p.pipes[i], params)
	}
	return p.outChan
}
