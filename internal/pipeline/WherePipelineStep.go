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
	"strings"

	"github.com/jackbister/logsuck/internal/events"
)

type wherePipelineStep struct {
	fieldValues map[string]string
}

func (s *wherePipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.input:
			if !ok {
				return
			}
			retEvents := make([]events.EventWithExtractedFields, 0, len(res.Events))
			for _, evt := range res.Events {
				include := true
				for k, v := range s.fieldValues {
					actualValue := evt.Fields[strings.ToLower(k)]
					if actualValue != v {
						include = false
					}
				}
				if include {
					retEvents = append(retEvents, evt)
				}
			}
			res.Events = retEvents

			retTableRows := make([]map[string]string, 0, len(res.TableRows))
			for _, tr := range res.TableRows {
				include := true
				for k, v := range s.fieldValues {
					actualValue := tr[k]
					if actualValue != v {
						include = false
					}
				}
				if include {
					retTableRows = append(retTableRows, tr)
				}
			}
			res.TableRows = retTableRows
			pipe.output <- res
		}
	}
}

func (s *wherePipelineStep) Name() string {
	return "where"
}

func (r *wherePipelineStep) InputType() PipelinePipeType {
	return PipelinePipeTypePropagate
}

func (r *wherePipelineStep) OutputType() PipelinePipeType {
	return PipelinePipeTypePropagate
}

func compileWhereStep(input string, options map[string]string) (pipelineStep, error) {
	return &wherePipelineStep{
		fieldValues: options,
	}, nil
}
