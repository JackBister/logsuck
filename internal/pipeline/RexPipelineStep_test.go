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

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
)

func TestRexPipelineStep(t *testing.T) {
	rps, err := compileRexStep("userid was (?P<userid>\\d+).", map[string]string{})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	_, dc := newDynamicConfig()
	params := PipelineParameters{
		Cfg:           &config.StaticConfig{},
		DynamicConfig: dc,
		EventsRepo:    repo,
	}
	pipe, input, output := newPipe()

	go rps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id:        1,
				Fields:    map[string]string{},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	evt := verifyEvt(t, output)
	verifyField(t, evt, "userid", "123")
}

func TestRexPipelineStep_MultipleExtractions(t *testing.T) {
	rps, err := compileRexStep("(\\w+)=(\\w+)", map[string]string{})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	_, dc := newDynamicConfig()
	params := PipelineParameters{
		Cfg:           &config.StaticConfig{},
		DynamicConfig: dc,
		EventsRepo:    repo,
	}
	pipe, input, output := newPipe()

	go rps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id:        1,
				Fields:    map[string]string{},
				Raw:       "2021-01-20 19:37:00 The user did something. userid=123, thingid=456.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	evt := verifyEvt(t, output)
	verifyField(t, evt, "userid", "123")
	verifyField(t, evt, "thingid", "456")
}

func TestRexPipelineStep_ExtractedField(t *testing.T) {
	rps, err := compileRexStep("userid was (?P<userid>\\d+).", map[string]string{
		"field": "MyExtractedField",
	})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	_, dc := newDynamicConfig()
	params := PipelineParameters{
		Cfg:           &config.StaticConfig{},
		DynamicConfig: dc,
		EventsRepo:    repo,
	}
	pipe, input, output := newPipe()

	go rps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id: 1,
				Fields: map[string]string{
					"MyExtractedField": "The userid was 123.",
				},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123.",
				Host:      "my-host",
				Source:    "my-log.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	evt := verifyEvt(t, output)
	verifyField(t, evt, "userid", "123")
}

func TestRexPipelineStep_Source(t *testing.T) {
	rps, err := compileRexStep("log-(?P<logid>\\d+)", map[string]string{
		"field": "source",
	})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	_, dc := newDynamicConfig()
	params := PipelineParameters{
		Cfg:           &config.StaticConfig{},
		DynamicConfig: dc,
		EventsRepo:    repo,
	}
	pipe, input, output := newPipe()

	go rps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id:        1,
				Fields:    map[string]string{},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123.",
				Host:      "my-host",
				Source:    "log-123.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	evt := verifyEvt(t, output)
	verifyField(t, evt, "logid", "123")
}

func TestRexPipelineStep_Host(t *testing.T) {
	rps, err := compileRexStep("host-(?P<hostid>\\d+)", map[string]string{
		"field": "host",
	})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	_, dc := newDynamicConfig()
	params := PipelineParameters{
		Cfg:           &config.StaticConfig{},
		DynamicConfig: dc,
		EventsRepo:    repo,
	}
	pipe, input, output := newPipe()

	go rps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
		Events: []events.EventWithExtractedFields{
			{
				Id:        1,
				Fields:    map[string]string{},
				Raw:       "2021-01-20 19:37:00 The user did something. The userid was 123.",
				Host:      "host-123",
				Source:    "log-123.txt",
				SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
				Timestamp: time.Date(2021, 1, 20, 19, 37, 0, 0, time.UTC),
			},
		},
	}
	close(input)

	evt := verifyEvt(t, output)
	verifyField(t, evt, "hostid", "123")
}

func verifyEvt(t *testing.T, output chan PipelineStepResult) events.EventWithExtractedFields {
	result, ok := <-output
	if !ok {
		t.Fatal("TestRexPipelineStep got unexpected !ok when receiving output")
	}
	if len(result.Events) != 1 {
		t.Fatalf("TestRexPipelineStep got unexpected number of events, expected 1 but got %v", len(result.Events))
	}
	_, ok = <-output
	if ok {
		t.Fatal("TestRexPipelineStep got unexpected ok when receiving output, expected the channel to be closed by now")
	}
	evt := result.Events[0]

	return evt
}

func verifyField(t *testing.T, evt events.EventWithExtractedFields, fieldName, expectedFieldValue string) {
	actualFieldValue, ok := evt.Fields[fieldName]
	if !ok {
		t.Fatal("TestRexPipelineStep got unexpected !ok when getting extracted field from fields map")
	}
	if actualFieldValue != expectedFieldValue {
		t.Fatalf("TestRexPipelineStep got unexpected field value, expected '%v' but got '%v'", expectedFieldValue, actualFieldValue)
	}
}
