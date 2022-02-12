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

package tasks

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackbister/logsuck/internal/config"
)

type Task interface {
	Name() string
	Run(cfg config.DynamicConfig, ctx context.Context)
}

type TaskState int

const (
	TaskStateNotRunning TaskState = 0
	TaskStateRunning    TaskState = 1
)

type TaskData struct {
	enabled    bool
	interval   time.Duration
	cfg        config.DynamicConfig
	state      TaskState
	ctx        context.Context
	cancelFunc context.CancelFunc
}

type TaskManager struct {
	cfg      *config.TasksConfig
	tasks    map[string]Task
	taskData sync.Map //<string, TaskData>
	ctx      context.Context
}

func NewTaskManager(cfg *config.TasksConfig, ctx context.Context) *TaskManager {
	return &TaskManager{
		cfg:      cfg,
		tasks:    map[string]Task{},
		taskData: sync.Map{},
		ctx:      ctx,
	}
}

func (tm *TaskManager) AddTask(t Task) error {
	name := t.Name()
	if _, exists := tm.tasks[name]; exists {
		return fmt.Errorf("a task with name=%s already exists", t.Name())
	}
	tm.tasks[name] = t

	ctx, cancelFunc := context.WithCancel(tm.ctx)
	var td TaskData
	if cfg, ok := tm.cfg.Tasks[name]; ok {
		td = TaskData{
			enabled:    cfg.Enabled,
			interval:   cfg.Interval,
			cfg:        cfg.Config,
			state:      TaskStateNotRunning,
			ctx:        ctx,
			cancelFunc: cancelFunc,
		}
	} else {
		td = TaskData{
			enabled:    false,
			interval:   1 * time.Hour,
			cfg:        config.NewDynamicConfig([]config.ConfigSource{}),
			state:      TaskStateNotRunning,
			ctx:        ctx,
			cancelFunc: cancelFunc,
		}
	}
	tm.taskData.Store(name, td)
	err := tm.ScheduleTask(name)
	if err != nil {
		return fmt.Errorf("failed to schedule task='%s': %w", name, err)
	}
	return nil
}

func (tm *TaskManager) ScheduleTask(name string) error {
	t, ok := tm.tasks[name]
	if !ok {
		return fmt.Errorf("task with name='%s' not found", name)
	}
	tdInterface, ok := tm.taskData.Load(name)
	if !ok {
		return fmt.Errorf("taskData for task='%s' not found", name)
	}
	td, ok := tdInterface.(TaskData)
	if !ok {
		return fmt.Errorf("failed to cast taskData for task='%s', taskData=%v", name, tdInterface)
	}
	log.Printf("scheduling task='%s' with interval=%v\n", name, td.interval)
	go func(t Task, td TaskData) {
		ticker := time.NewTicker(td.interval)
		defer ticker.Stop()

		name := t.Name()
		for {
			select {
			case <-td.ctx.Done():
				log.Printf("context cancelled for task='%s'\n", name)
				return
			case <-ticker.C:
				if !td.enabled {
					log.Printf("not running task='%s' because it is disabled", name)
				} else {
					log.Printf("running task='%s'\n", name)
					startTime := time.Now()
					td.state = TaskStateRunning
					tm.taskData.Store(name, td)
					t.Run(td.cfg, td.ctx)
					endTime := time.Now()
					td.state = TaskStateNotRunning
					tm.taskData.Store(name, td)
					log.Printf("task='%s' finished in time=%v", name, endTime.Sub(startTime))
				}
			}
		}
	}(t, td)
	return nil
}
