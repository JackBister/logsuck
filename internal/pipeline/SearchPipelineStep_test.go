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

	"github.com/jackbister/logsuck/internal/events"
)

func TestSearchPipelineStep(t *testing.T) {
	sps, err := compileSearchStep("", map[string]string{})
	if err != nil {
		t.Fatalf("TestSearchPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	params := PipelineParameters{
		ConfigSource: newConfigSource(),
		EventsRepo:   repo,
	}
	pipe, input, output := newPipe()
	close(input)
	repo.AddBatch([]events.Event{
		{
			Raw:       "2021-01-20 20:29:00 This is an event",
			Host:      "MYHOST",
			Offset:    0,
			Source:    "my-log.txt",
			SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
			Timestamp: time.Date(2021, 1, 20, 20, 29, 0, 0, time.UTC),
		},
	})

	go sps.Execute(context.Background(), pipe, params)

	result, ok := <-output
	if !ok {
		t.Fatal("TestSearchPipelineStep got unexpected !ok when receiving output")
	}
	if len(result.Events) != 1 {
		t.Fatalf("TestSearchPipelineStep got unexpected number of events, expected 1 but got %v", len(result.Events))
	}
	_, ok = <-output
	if ok {
		t.Fatal("TestSearchPipelineStep got unexpected ok when receiving output, expected the channel to be closed by now")
	}
}
