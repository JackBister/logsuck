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

package steps

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

func TestWherePipelineStep(t *testing.T) {
	input, output := setup(t, map[string]string{"userId": "123"})
	input <- pipeline.StepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id: 1,
				Fields: map[string]string{
					"userid": "123",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
			{
				Id: 2,
				Fields: map[string]string{
					"userid": "456",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 456.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	result, ok := <-output
	if !ok {
		t.Fatal("got unexpected !ok when receiving output")
	}
	if len(result.Events) != 1 {
		t.Fatalf("got unexpected number of events, expected 1 but got %v", len(result.Events))
	}
	_, ok = <-output
	if ok {
		t.Fatal("got unexpected ok when receiving output, expected the channel to be closed by now")
	}
}

func TestWherePipelineStep_MultipleConditions(t *testing.T) {
	input, output := setup(t, map[string]string{"userId": "123", "username": "charles"})
	input <- pipeline.StepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id: 1,
				Fields: map[string]string{
					"userid":   "123",
					"username": "charles",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123. The username was charles.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
			{
				Id: 2,
				Fields: map[string]string{
					"userid":   "456",
					"username": "jonny",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 456. The username was jonny.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
			{
				Id: 3,
				Fields: map[string]string{
					"userid": "789",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 789. The username was unknown.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	result, ok := <-output
	if !ok {
		t.Fatal("got unexpected !ok when receiving output")
	}
	if len(result.Events) != 1 {
		t.Fatalf("got unexpected number of events, expected 1 but got %v", len(result.Events))
	}
	_, ok = <-output
	if ok {
		t.Fatal("got unexpected ok when receiving output, expected the channel to be closed by now")
	}
}

func TestWherePipelineStep_TableInput(t *testing.T) {
	input, output := setup(t, map[string]string{"userId": "123"})
	input <- pipeline.StepResult{
		TableRows: []map[string]string{
			{
				"userId":   "123",
				"username": "charles",
			},
			{
				"userId":   "456",
				"username": "jonny",
			},
		},
	}
	close(input)

	result, ok := <-output
	if !ok {
		t.Fatal("got unexpected !ok when receiving output")
	}
	if len(result.TableRows) != 1 {
		t.Fatalf("got unexpected number of table rows, expected 1 but got %v", len(result.TableRows))
	}
	_, ok = <-output
	if ok {
		t.Fatal("got unexpected ok when receiving output, expected the channel to be closed by now")
	}
}

func setup(t *testing.T, fieldValues map[string]string) (input chan pipeline.StepResult, output chan pipeline.StepResult) {
	wps, err := compileWhereStep("", fieldValues)
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	params := pipeline.Parameters{
		ConfigSource: &config.NullSource{},
		EventsRepo:   repo,

		Logger: slog.Default(),
	}
	pipe, input, output := newPipe()

	go wps.Execute(context.Background(), pipe, params)

	return input, output
}
