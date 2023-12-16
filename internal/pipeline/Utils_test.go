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
	"log/slog"

	api "github.com/jackbister/logsuck/pkg/logsuck/pipeline"
	"go.uber.org/dig"

	"github.com/jackbister/logsuck/plugins/steps"
)

func newTestPipelineCompiler() *PipelineCompiler {
	c := dig.New()
	err := steps.Plugin.Provide(c, slog.Default())
	if err != nil {
		panic(err)
	}
	var ret PipelineCompiler
	err = c.Invoke(func(p struct {
		dig.In

		StepDefinitions []api.StepDefinition `group:"steps"`
	}) {
		m := map[string]api.StepDefinition{}
		for _, v := range p.StepDefinitions {
			m[v.StepName] = v
		}
		ret = PipelineCompiler{
			stepDefinitions: m,
		}
	})
	if err != nil {
		panic(err)
	}
	return &ret
}
