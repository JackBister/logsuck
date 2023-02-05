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
	"sync"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"go.uber.org/zap"
)

type TaskContext struct {
	EventsRepo events.Repository
}

type Task interface {
	Name() string
	Run(cfg map[string]any, ctx context.Context)
}

type TaskConstructor func(t TaskContext, logger *zap.Logger) Task

var taskMap = map[string]TaskConstructor{
	"@logsuck/DeleteOldEventsTask": func(t TaskContext, logger *zap.Logger) Task {
		return &DeleteOldEventsTask{Repo: t.EventsRepo, Logger: logger}
	},
}

type TaskState int

const (
	TaskStateNotRunning TaskState = 0
	TaskStateRunning    TaskState = 1
)

type TaskData struct {
	enabled     bool
	interval    time.Duration
	cfg         map[string]any
	state       TaskState
	ctx         context.Context
	cancelFunc  context.CancelFunc
	isCancelled chan struct{}
}

type TaskManager struct {
	cfg         *config.TasksConfig
	taskContext TaskContext
	tasks       map[string]Task
	taskData    sync.Map //<string, TaskData>
	ctx         context.Context

	logger *zap.Logger
}

func NewTaskManager(cfg *config.TasksConfig, taskContext TaskContext, ctx context.Context, logger *zap.Logger) *TaskManager {
	return &TaskManager{
		cfg:         cfg,
		taskContext: taskContext,
		tasks:       map[string]Task{},
		taskData:    sync.Map{},
		ctx:         ctx,

		logger: logger,
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
			enabled:     cfg.Enabled,
			interval:    cfg.Interval,
			cfg:         cfg.Config,
			state:       TaskStateNotRunning,
			ctx:         ctx,
			cancelFunc:  cancelFunc,
			isCancelled: make(chan struct{}, 1),
		}
	} else {
		td = TaskData{
			enabled:     false,
			interval:    1 * time.Hour,
			cfg:         map[string]any{},
			state:       TaskStateNotRunning,
			ctx:         ctx,
			cancelFunc:  cancelFunc,
			isCancelled: make(chan struct{}, 1),
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
	logger := tm.logger.With(zap.String("taskName", name))
	logger.Info("scheduling task",
		zap.Stringer("interval", td.interval))
	go func(t Task, td TaskData) {
		ticker := time.NewTicker(td.interval)
		defer ticker.Stop()

		name := t.Name()
		for {
			select {
			case <-td.ctx.Done():
				logger.Info("context cancelled for task")
				td.isCancelled <- struct{}{}
				return
			case <-ticker.C:
				if !td.enabled {
					logger.Info("not running task because it is disabled")
				} else {
					logger.Info("running task")
					startTime := time.Now()
					td.state = TaskStateRunning
					tm.taskData.Store(name, td)
					t.Run(td.cfg, td.ctx)
					endTime := time.Now()
					td.state = TaskStateNotRunning
					tm.taskData.Store(name, td)
					logger.Info("task finished",
						zap.Stringer("duration", endTime.Sub(startTime)))
				}
			}
		}
	}(t, td)
	return nil
}

func (tm *TaskManager) UpdateConfig(cfg config.Config) {
	tm.logger.Info("Updating TaskManager config")
	tm.cfg = &cfg.Tasks
	for tn := range tm.tasks {
		logger := tm.logger.With(zap.String("taskName", tn))
		oldTdAny, ok := tm.taskData.Load(tn)
		if !ok {
			logger.Error("did not get taskData for task")
			continue
		}
		oldTd, ok := oldTdAny.(TaskData)
		if !ok {
			logger.Error("failed to cast taskData for task")
			continue
		}
		oldTd.cancelFunc()
		<-oldTd.isCancelled
	}

	tasksToRemove := []string{}
	for tn := range tm.tasks {
		if t, ok := cfg.Tasks.Tasks[tn]; !ok || !t.Enabled {
			tm.logger.Info("task was not present in new configuration. This task will not be rescheduled",
				zap.String("taskName", tn))
			tasksToRemove = append(tasksToRemove, tn)
		}
	}

	for _, tn := range tasksToRemove {
		tm.taskData.Delete(tn)
		delete(tm.tasks, tn)
	}

	for _, t := range cfg.Tasks.Tasks {
		if !t.Enabled {
			continue
		}
		taskConstructor, ok := taskMap[t.Name]
		if !ok {
			tm.logger.Error("did not find taskConstructor, this task will be ignored",
				zap.String("taskName", t.Name))
			continue
		}
		task := taskConstructor(tm.taskContext, tm.logger.Named(t.Name))
		tm.taskData.Delete(t.Name)
		delete(tm.tasks, t.Name)
		tm.AddTask(task)
	}
}
