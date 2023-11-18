package tasks

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewDeleteOldEventsTask, dig.Group("tasks"))
		if err != nil {
			return err
		}
		return nil
	},
}
