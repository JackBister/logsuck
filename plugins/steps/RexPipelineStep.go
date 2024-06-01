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
	"fmt"
	"regexp"

	"github.com/jackbister/logsuck/pkg/logsuck/parser"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type RexPipelineStep struct {
	extractor regexp.Regexp
	field     string
}

func (r *RexPipelineStep) Execute(ctx context.Context, pipe pipeline.Pipe, params pipeline.Parameters) {
	defer close(pipe.Output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.Input:
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
					Cfg: config.RegexParserConfig{
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
			pipe.Output <- res
		}
	}
}

func (r *RexPipelineStep) Name() string {
	return "rex"
}

func (r *RexPipelineStep) InputType() pipeline.PipeType {
	return pipeline.PipeTypeEvents
}

func (r *RexPipelineStep) OutputType() pipeline.PipeType {
	return pipeline.PipeTypeEvents
}

func compileRexStep(input string, options map[string]string) (pipeline.Step, error) {
	field, ok := options["field"]
	if !ok {
		field = "_raw"
	}

	regex, err := regexp.Compile(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compile rex: %w", err)
	}

	return &RexPipelineStep{
		extractor: *regex,
		field:     field,
	}, nil
}
