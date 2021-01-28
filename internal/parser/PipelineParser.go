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

package parser

import (
	"fmt"
	"strings"
)

type ParsedPipelineStep struct {
	StepType string
	Args     map[string]string
	Value    string
}

type PipelineParseResult struct {
	Steps []ParsedPipelineStep
}

func ParsePipeline(s string) (*PipelineParseResult, error) {
	tokens, err := tokenize(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	p := parser{
		tokens: tokens,
	}

	steps := make([]ParsedPipelineStep, 0)

	// If the first token is not a pipe, all tokens up to the first pipe are used as the value for a search step
	if p.peek() != tokenPipe {
		sb := strings.Builder{}
		for len(p.tokens) > 0 && p.peek() != tokenPipe {
			sb.WriteString(p.take().value)
		}
		steps = append(steps, ParsedPipelineStep{
			StepType: "search",
			Args:     map[string]string{},
			Value:    sb.String(),
		})
	}

	for len(p.tokens) > 0 {
		step := ParsedPipelineStep{
			Args: map[string]string{},
		}
		p.skipWhitespace()
		_, err = p.require(tokenPipe)
		if err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
		p.skipWhitespace()
		tokStepType, err := p.require(tokenString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
		step.StepType = tokStepType.value
		p.skipWhitespace()
		if p.peek() == tokenString {
			key := p.take().value
			p.skipWhitespace()
			_, err := p.require(tokenEquals)
			if err != nil {
				return nil, fmt.Errorf("failed to parse: %w", err)
			}
			p.skipWhitespace()
			if p.peek() != tokenString && p.peek() != tokenQuotedString {
				return nil, fmt.Errorf("failed to parse: expected string or quoted string in option list for command %v", step.StepType)
			}
			tokFieldValue := p.take()
			step.Args[key] = tokFieldValue.value
			p.skipWhitespace()
		}
		if p.peek() == tokenQuotedString || p.peek() == tokenString {
			step.Value = p.take().value
		} else {
			step.Value = ""
		}
		steps = append(steps, step)
	}

	return &PipelineParseResult{
		Steps: steps,
	}, nil
}
