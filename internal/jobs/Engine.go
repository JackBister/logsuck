// Copyright 2021 Jack Bister
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

package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	internalPipeline "github.com/jackbister/logsuck/internal/pipeline"
	"github.com/jackbister/logsuck/pkg/logsuck/events"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	api "github.com/jackbister/logsuck/pkg/logsuck/jobs"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"

	"go.uber.org/dig"
)

type Engine struct {
	cancels      map[int64]func()
	configSource config.ConfigSource
	eventRepo    events.Repository
	jobRepo      api.Repository

	logger *slog.Logger
}

type EngineParams struct {
	dig.In

	ConfigSource config.ConfigSource
	EventRepo    events.Repository
	JobRepo      api.Repository
	Logger       *slog.Logger
}

func NewEngine(p EngineParams) *Engine {
	return &Engine{
		cancels:      map[int64]func(){},
		configSource: p.ConfigSource,
		eventRepo:    p.EventRepo,
		jobRepo:      p.JobRepo,

		logger: p.Logger,
	}
}

func (e *Engine) StartJob(query string, startTime, endTime *time.Time) (*int64, error) {
	pl, err := internalPipeline.CompilePipeline(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to compile search query: %w", err)
	}
	sortMode := pl.SortMode()
	columnOrder, err := pl.ColumnOrder()
	if err != nil {
		e.logger.Warn("Got error when getting column order for pipeline. The columns may not be ordered correctly when displayed.", slog.Any("error", err))
		columnOrder = []string{}
	}
	id, err := e.jobRepo.Insert(query, startTime, endTime, sortMode, pl.OutputType(), columnOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job in repo: %w", err)
	}
	logger := e.logger.With(slog.Int64("jobId", *id))
	ctx, cancelFunc := context.WithCancel(context.Background())
	e.cancels[*id] = cancelFunc
	go func() {
		done := ctx.Done()
		outputType := pl.OutputType()
		rowNumber := 0
		// TODO: This should probably be batched
		results := pl.Execute(
			ctx,
			internalPipeline.PipelineParameters{
				ConfigSource: e.configSource,
				EventsRepo:   e.eventRepo,

				Logger: logger,
			})
		wasCancelled := false
	out:
		for {
			select {
			case res, ok := <-results:
				if !ok {
					break out
				}
				if outputType == pipeline.PipelinePipeTypeEvents {
					evts := res.Events
					if len(evts) > 0 {
						converted := make([]events.EventIdAndTimestamp, len(evts))
						for i, evt := range evts {
							converted[i] = events.EventIdAndTimestamp{
								Id:        evt.Id,
								Timestamp: evt.Timestamp,
							}
						}
						err := e.jobRepo.AddResults(*id, converted)
						if err != nil {
							logger.Error("failed to add events to job",
								slog.Any("error", err))
							// TODO: Retry?
							continue
						}
						fields := gatherFieldStats(evts)
						err = e.jobRepo.AddFieldStats(*id, fields)
						if err != nil {
							logger.Error("failed to add field stats to job",
								slog.Any("error", err))
						}
					}
				} else if outputType == pipeline.PipelinePipeTypeTable {
					tableRows := res.TableRows
					if len(tableRows) > 0 {
						rows := make([]api.TableRow, 0, len(tableRows))
						for _, r := range tableRows {
							rows = append(rows, api.TableRow{
								RowNumber: rowNumber,
								Values:    r,
							})
							rowNumber++
						}
						err := e.jobRepo.AddTableResults(*id, rows)
						if err != nil {
							logger.Error("failed to add table results to job",
								slog.Any("error", err))
							// TODO: Retry?
							continue
						}
						fields := gatherFieldStatsFromTable(tableRows)
						err = e.jobRepo.AddFieldStats(*id, fields)
						if err != nil {
							logger.Error("failed to add field stats to job",
								slog.Any("error", err))
						}
					}
				} else {
					logger.Error("unhandled outputType", slog.Int("outputType", int(outputType)))
				}
			case <-done:
				wasCancelled = true
				break out
			}
		}
		e.cancels[*id] = nil
		var state api.JobState
		if wasCancelled {
			state = api.JobStateAborted
		} else {
			state = api.JobStateFinished
		}
		err = e.jobRepo.UpdateState(*id, state)
		if err != nil {
			logger.Error("failed to update job to finished state",
				slog.Any("error", err))
		}
	}()
	return id, nil
}

func (e *Engine) Abort(jobId int64) error {
	cancelFunc := e.cancels[jobId]
	if cancelFunc != nil {
		cancelFunc()
		return nil
	}
	logger := e.logger.With(slog.Int64("jobId", jobId))
	logger.Warn("Attempted to cancel job but there was no cancelFunc in the cancels map. Will verify that state is aborted or finished")
	job, err := e.jobRepo.Get(jobId)
	if err != nil {
		logger.Error("Got error when verifying that job is aborted or finished. The job is in an unknown state.")
		return errors.New("job does not appear to be running, but the state in the repository could not be verified")
	}
	if job.State == api.JobStateRunning {
		logger.Error("job has no entry in the cancels map, but state is running. Will set state to aborted. This may signify that there is a bug and the job may actually still be running.")
		err = e.jobRepo.UpdateState(jobId, api.JobStateAborted)
		if err != nil {
			return errors.New("job does not appear to be running, but the state in the repository could not be set to aborted")
		}
	}
	return nil
}

func gatherFieldStats(evts []events.EventWithExtractedFields) []api.FieldStats {
	m := map[string]map[string]int{}
	size := 0
	for _, evt := range evts {
		for k, v := range evt.Fields {
			if _, ok := m[k]; !ok {
				size++
				m[k] = map[string]int{}
				m[k][v] = 1
			} else if _, ok := m[k][v]; !ok {
				m[k][v] = 1
			} else {
				m[k][v]++
			}
		}
	}

	ret := make([]api.FieldStats, 0, size)
	for k, vm := range m {
		for v, o := range vm {
			ret = append(ret, api.FieldStats{
				Key:         k,
				Value:       v,
				Occurrences: o,
			})
		}
	}
	return ret
}

func gatherFieldStatsFromTable(rows []map[string]string) []api.FieldStats {
	m := map[string]map[string]int{}
	size := 0
	for _, row := range rows {
		for k, v := range row {
			if _, ok := m[k]; !ok {
				size++
				m[k] = map[string]int{}
				m[k][v] = 1
			} else if _, ok := m[k][v]; !ok {
				m[k][v] = 1
			} else {
				m[k][v]++
			}
		}
	}

	ret := make([]api.FieldStats, 0, size)
	for k, vm := range m {
		for v, o := range vm {
			ret = append(ret, api.FieldStats{
				Key:         k,
				Value:       v,
				Occurrences: o,
			})
		}
	}
	return ret
}
