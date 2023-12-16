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
