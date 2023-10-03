// Copyright 2023 Jack Bister
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
	"fmt"
	"strings"

	api "github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type tablePipelineStep struct {
	fields []string
}

func (s *tablePipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.input:
			if !ok {
				return
			}
			ret := make([]map[string]string, 0)
			for _, evt := range res.Events {
				m := map[string]string{}
				for _, f := range s.fields {
					m[f] = evt.Fields[f]
				}
				ret = append(ret, m)
			}
			res.TableRows = ret
			pipe.output <- res
		}
	}
}

func (s *tablePipelineStep) ColumnOrder() []string {
	return s.fields
}

func (s *tablePipelineStep) Name() string {
	return "table"
}

func (r *tablePipelineStep) InputType() api.PipelinePipeType {
	return api.PipelinePipeTypeEvents
}

func (r *tablePipelineStep) OutputType() api.PipelinePipeType {
	return api.PipelinePipeTypeTable
}

func compileTableStep(input string, options map[string]string) (pipelineStep, error) {
	fields := strings.Split(input, ",")
	trimmedFields := make([]string, 0, len(fields))
	for _, f := range fields {
		trimmed := strings.TrimSpace(f)
		if len(trimmed) > 0 {
			trimmedFields = append(trimmedFields, trimmed)
		}
	}
	if len(trimmedFields) == 0 {
		return nil, fmt.Errorf("failed to compile table: no fields given. You must specify which fields should be present in the table using this syntax: '| table \"field1, field2, ...\"'")
	}
	return &tablePipelineStep{
		fields: trimmedFields,
	}, nil
}
