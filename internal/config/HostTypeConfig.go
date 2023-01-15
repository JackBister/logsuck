package config

import (
	"time"
)

const defaultReadInterval = 1 * time.Second

type HostFileConfig struct {
	Name string
}

type HostTypeConfig struct {
	Files []HostFileConfig
}
