// Copyright 2021 The Logsuck Authors
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
			ret := make([]events.EventWithExtractedFields, 0, len(res.Events))
			for _, evt := range res.Events {
				include := true
				for k, v := range s.fieldValues {
					actualValue := evt.Fields[k]
					if actualValue != v {
						include = false
					}
				}
				if include {
					ret = append(ret, evt)
				}
			}
			res.Events = ret
			pipe.output <- res
		}
	}
}

func compileWhereStep(input string, options map[string]string) (pipelineStep, error) {
	// This is pretty hacky. I think every command type might need to do its own parsing in the future.
	m := make(map[string]string, len(options))
	for k, v := range options {
		m[strings.ToLower(k)] = v
	}
	return &wherePipelineStep{
		fieldValues: m,
	}, nil
}
