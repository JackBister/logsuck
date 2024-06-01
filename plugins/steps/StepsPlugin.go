// Copyright 2024 Jack Bister
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

package steps

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/steps",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func() pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "rex",
				Compiler: compileRexStep,
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		err = c.Provide(func() pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "search",
				Compiler: compileSearchStep,
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		err = c.Provide(func() pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "surrounding",
				Compiler: compileSurroundingStep,
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		err = c.Provide(func() pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "table",
				Compiler: compileTableStep,
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		err = c.Provide(func() pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "where",
				Compiler: compileWhereStep,
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		return nil
	},
}
