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
	"regexp"

	"github.com/jackbister/logsuck/internal/parser"
)

type rexPipelineStep struct {
	extractor regexp.Regexp
	field     string
}

func (r *rexPipelineStep) Execute(ctx context.Context, pipe pipelinePipe, params PipelineParameters) {
	defer close(pipe.output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.input:
			if !ok {
				return
			}
			for _, evt := range res.Events {
				fieldValue, ok := evt.Fields[r.field]
				if r.field == "_raw" {
					fieldValue = evt.Raw
				} else if r.field == "source" {
					fieldValue = evt.Source
				} else if r.field == "host" {
					fieldValue = evt.Host
				} else if !ok {
					continue // Maybe this should be logged or put in some kind of metrics
				}
				p := parser.RegexFileParser{
					Cfg: parser.RegexParserConfig{
						EventDelimiter: regexp.MustCompile("\n"),
						FieldExtractors: []*regexp.Regexp{
							&r.extractor,
						},
					},
				}
				newFields, _ := parser.ExtractFields(fieldValue, &p)
				for k, v := range newFields {
					// Is mutating the event in place like this dangerous?
					// I don't think so since the events are paid forward through channels so only one step should touch them at a time,
					// and this avoids an extra allocation for each batch+step combo
					evt.Fields[k] = v
				}
			}
			pipe.output <- res
		}
	}
}

func (r *rexPipelineStep) Name() string {
	return "rex"
}

func (r *rexPipelineStep) InputType() PipelinePipeType {
	return PipelinePipeTypeEvents
}

func (r *rexPipelineStep) OutputType() PipelinePipeType {
	return PipelinePipeTypeEvents
}

func compileRexStep(input string, options map[string]string) (pipelineStep, error) {
	field, ok := options["field"]
	if !ok {
		field = "_raw"
	}

	regex, err := regexp.Compile(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile rex: %w", err)
	}

	return &rexPipelineStep{
		extractor: *regex,
		field:     field,
	}, nil
}
