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
	"log/slog"
	"sync"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/web"
	"go.uber.org/dig"
)

type TaskContext struct {
	EventsRepo events.Repository
}

type Task interface {
	Name() string
	Run(cfg map[string]any, ctx context.Context)
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

type TaskEnumProvider struct {
	tm *TaskManager
}

func NewTaskEnumProvider(tm *TaskManager) web.EnumProvider {
	return &TaskEnumProvider{tm: tm}
}

func (te *TaskEnumProvider) Name() string {
	return "tasks"
}

func (te *TaskEnumProvider) Values() ([]string, error) {
	ret := make([]string, 0, len(te.tm.tasks))
	for k := range te.tm.tasks {
		ret = append(ret, k)
	}
	return ret, nil
}

type TaskManager struct {
	cfg         *config.TasksConfig
	taskContext TaskContext
	tasks       map[string]Task
	taskData    sync.Map //<string, TaskData>
	ctx         context.Context

	logger *slog.Logger
}

type TaskManagerParams struct {
	dig.In

	EventsRepo events.Repository
	Ctx        context.Context
	CfgSource  config.ConfigSource
	Logger     *slog.Logger

	Tasks []Task `group:"tasks"`
}

func NewTaskManager(p TaskManagerParams) (*TaskManager, error) {
	r, err := p.CfgSource.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to create TaskManager: %w", err)
	}

	tm := &TaskManager{
		cfg: &r.Cfg.Tasks,
		taskContext: TaskContext{
			EventsRepo: p.EventsRepo,
		},
		tasks:    map[string]Task{},
		taskData: sync.Map{},
		ctx:      p.Ctx,

		logger: p.Logger,
	}
	for _, t := range p.Tasks {
		tm.tasks[t.Name()] = t
	}
	tm.UpdateConfig(r.Cfg)
	return tm, nil
}

func (tm *TaskManager) ScheduleTask(name string) error {
	t, ok := tm.tasks[name]
	if !ok {
		return fmt.Errorf("task with name='%s' not found", name)
	}

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
	logger := tm.logger.With(slog.String("taskName", name))
	logger.Info("scheduling task",
		slog.Duration("interval", td.interval))
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
						slog.Duration("duration", endTime.Sub(startTime)))
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
		logger := tm.logger.With(slog.String("taskName", tn))
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
		if oldTd.isCancelled != nil {
			<-oldTd.isCancelled
		}
	}

	tasksToRemove := []string{}
	for tn := range tm.tasks {
		if t, ok := cfg.Tasks.Tasks[tn]; !ok || !t.Enabled {
			tm.logger.Info("task was not present in new configuration. This task will not be rescheduled",
				slog.String("taskName", tn))
			tasksToRemove = append(tasksToRemove, tn)
		}
	}

	for _, tn := range tasksToRemove {
		tm.taskData.Delete(tn)
	}

	for _, t := range cfg.Tasks.Tasks {
		if !t.Enabled {
			continue
		}
		tm.ScheduleTask(t.Name)
	}
}
