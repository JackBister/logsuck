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

package steps

import (
	"context"
	"strings"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	api "github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type WherePipelineStep struct {
	fieldValues map[string]string
}

func (s *WherePipelineStep) Execute(ctx context.Context, pipe api.PipelinePipe, params api.PipelineParameters) {
	defer close(pipe.Output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.Input:
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
			pipe.Output <- res
		}
	}
}

func (s *WherePipelineStep) Name() string {
	return "where"
}

func (r *WherePipelineStep) InputType() api.PipelinePipeType {
	return api.PipelinePipeTypePropagate
}

func (r *WherePipelineStep) OutputType() api.PipelinePipeType {
	return api.PipelinePipeTypePropagate
}

func compileWhereStep(input string, options map[string]string) (api.PipelineStep, error) {
	return &WherePipelineStep{
		fieldValues: options,
	}, nil
}
