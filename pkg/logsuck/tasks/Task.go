package tasks

import "context"

type Task interface {
	Name() string
	Run(cfg map[string]any, ctx context.Context)
	ConfigSchema() map[string]any
}
