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
	"log"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/pipeline"
)

type Engine struct {
	cancels       map[int64]func()
	cfg           *config.StaticConfig
	dynamicConfig config.DynamicConfig
	eventRepo     events.Repository
	jobRepo       Repository
}

func NewEngine(cfg *config.StaticConfig, dynamicConfig config.DynamicConfig, eventRepo events.Repository, jobRepo Repository) *Engine {
	return &Engine{
		cancels:       map[int64]func(){},
		cfg:           cfg,
		dynamicConfig: dynamicConfig,
		eventRepo:     eventRepo,
		jobRepo:       jobRepo,
	}
}

func (e *Engine) StartJob(query string, startTime, endTime *time.Time) (*int64, error) {
	pl, err := pipeline.CompilePipeline(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to compile search query: %w", err)
	}
	sortMode := getSortMode(pl)
	id, err := e.jobRepo.Insert(query, startTime, endTime, sortMode)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job in repo: %w", err)
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	e.cancels[*id] = cancelFunc
	go func() {
		done := ctx.Done()
		// TODO: This should probably be batched
		results := pl.Execute(
			ctx,
			pipeline.PipelineParameters{
				Cfg:           e.cfg,
				DynamicConfig: e.dynamicConfig,
				EventsRepo:    e.eventRepo,
			})
		wasCancelled := false
	out:
		for {
			select {
			case res, ok := <-results:
				if !ok {
					break out
				}
				evts := res.Events
				log.Println("got", len(evts), "matching events")
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
						log.Println("Failed to add events to jobId=" + strconv.FormatInt(*id, 10) + ", error: " + err.Error())
						// TODO: Retry?
						continue
					}
					fields := gatherFieldStats(evts)
					err = e.jobRepo.AddFieldStats(*id, fields)
					if err != nil {
						log.Println("Failed to add field stats to jobId=" + strconv.FormatInt(*id, 10) + ", error: " + err.Error())
					}
				}
			case <-done:
				log.Println("<-done")
				wasCancelled = true
				break out
			}
		}
		e.cancels[*id] = nil
		var state JobState
		if wasCancelled {
			state = JobStateAborted
		} else {
			state = JobStateFinished
		}
		err = e.jobRepo.UpdateState(*id, state)
		if err != nil {
			log.Println("Failed to update jobId=" + strconv.FormatInt(*id, 10) + " when updating to finished state. err=" + err.Error())
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
	log.Printf("Attempted to cancel jobId=%v but there was no cancelFunc in the cancels map. Will verify that state is aborted or finished.\n", jobId)
	job, err := e.jobRepo.Get(jobId)
	if err != nil {
		log.Printf("Got error when verifying that jobId=%v is aborted or finished. The job is in an unknown state.\n", jobId)
		return errors.New("job does not appear to be running, but the state in the repository could not be verified")
	}
	if job.State == JobStateRunning {
		log.Printf("jobId=%v has no entry in the cancels map, but state is running. Will set state to aborted. This may signify that there is a bug and the job may actually still be running.\n", jobId)
		err = e.jobRepo.UpdateState(jobId, JobStateAborted)
		if err != nil {
			return errors.New("job does not appear to be running, but the state in the repository could not be set to aborted")
		}
	}
	return nil
}

func gatherFieldStats(evts []events.EventWithExtractedFields) []FieldStats {
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

	ret := make([]FieldStats, 0, size)
	for k, vm := range m {
		for v, o := range vm {
			ret = append(ret, FieldStats{
				Key:         k,
				Value:       v,
				Occurrences: o,
			})
		}
	}
	return ret
}

func getSortMode(pl *pipeline.Pipeline) events.SortMode {
	sn := pl.GetStepNames()
	if len(sn) > 0 && sn[len(sn)-1] == "surrounding" {
		return events.SortModePreserveArgOrder
	}
	return events.SortModeTimestampDesc
}
