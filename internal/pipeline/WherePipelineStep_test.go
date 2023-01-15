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

func TestWherePipelineStep(t *testing.T) {
	wps, err := compileWhereStep("", map[string]string{"userId": "123"})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	params := PipelineParameters{
		ConfigSource: &config.NullConfigSource{},
		EventsRepo:   repo,
	}
	pipe, input, output := newPipe()

	go wps.Execute(context.Background(), pipe, params)

	input <- PipelineStepResult{
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
