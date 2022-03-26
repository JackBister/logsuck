// Copyright 2022 Jack Bister
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

package config

import (
	"log"
	"time"
)

type TaskConfig struct {
	Name     string
	Enabled  bool
	Interval time.Duration
	Config   DynamicConfig
}

type TasksConfig struct {
	Tasks map[string]TaskConfig
}

func ReadDynamicTasksConfig(dynamicConfig DynamicConfig) (*TasksConfig, error) {
	tasksCfg := dynamicConfig.Cd("tasks")
	tasksArr, _ := tasksCfg.GetArray("tasks", []any{}).Get()
	tasks := make(map[string]TaskConfig, len(tasksArr))
	for i, t := range tasksArr {
		taskConfig, ok := t.(map[string]any)
		if !ok {
			log.Printf("failed to convert task config at index=%v to map[string]any. this task will not be enabled.\n", i)
			continue
		}
		name, ok := getValue[string](taskConfig, "name")
		if !ok {
			log.Printf("failed to get name for task at index=%v. this task will not be enabled.\n", i)
			continue
		}
		enabled, ok := getValue[bool](taskConfig, "enabled")
		if !ok {
			log.Printf("failed to get enabled for task with name=%v. this task will not be enabled.\n", name)
			continue
		}
		intervalString, ok := getValue[string](taskConfig, "interval")
		if !ok {
			log.Printf("failed to get interval for task with name=%v. this task will not be enabled.\n", name)
			continue
		}
		interval, err := time.ParseDuration(intervalString)
		if err != nil {
			log.Printf("failed to parse interval for task with name=%v. this task will not be enabled: %v\n", name, err)
			continue
		}
		configMap, ok := getValue[map[string]any](taskConfig, "config")
		if !ok {
			log.Printf("failed to get config for task with name=%v. this task will not be enabled.\n", name)
			continue
		}
		cfg := NewDynamicConfig([]ConfigSource{
			NewMapConfigSource(configMap),
		})
		tasks[name] = TaskConfig{
			Name:     name,
			Enabled:  enabled,
			Interval: interval,
			Config:   cfg,
		}
	}
	return &TasksConfig{
		Tasks: tasks,
	}, nil
}

func getValue[T any](m map[string]any, k string) (T, bool) {
	var ret T
	v, ok := m[k]
	if !ok {
		return ret, false
	}
	ret, ok = v.(T)
	if !ok {
		return ret, false
	}
	return ret, ok
}
