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
	tasksArr, _ := tasksCfg.GetArray("tasks", []interface{}{}).Get()
	tasks := make(map[string]TaskConfig, len(tasksArr))
	for _, t := range tasksArr {
		_, ok := t.(map[string]interface{})
		log.Println("task", t, ok)
	}
	return &TasksConfig{
		Tasks: tasks,
	}, nil
}
