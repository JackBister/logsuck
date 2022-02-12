package config

import "time"

type TaskConfig struct {
	Name     string
	Enabled  bool
	Interval time.Duration
	Config   DynamicConfig
}

type TasksConfig struct {
	Tasks map[string]TaskConfig
}
