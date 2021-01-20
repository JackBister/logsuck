// Copyright 2020 The Logsuck Authors
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
	"database/sql"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"

	_ "github.com/mattn/go-sqlite3"
)

func TestSearchPipelineStep(t *testing.T) {
	sps, err := compileSearchStep("", map[string]string{})
	if err != nil {
		t.Fatalf("TestSearchPipelineStep got unexpected error: %v", err)
	}

	input := make(chan PipelineStepResult)
	close(input)
	output := make(chan PipelineStepResult)

	db, err := sql.Open("sqlite3", ":memory:")
	defer db.Close()
	if err != nil {
		t.Fatalf("TestSearchPipelineStep got error when creating in-memory SQLite database: %v", err)
	}
	repo, err := events.SqliteRepository(db)
	if err != nil {
		t.Fatalf("TestSearchPipelineStep got error when creating events repo: %v", err)
	}

	repo.AddBatch([]events.Event{
		{
			Raw:       "2021-01-20 20:29:00 This is an event",
			Host:      "MYHOST",
			Offset:    0,
			Source:    "my-log.txt",
			Timestamp: time.Date(2021, 1, 20, 20, 29, 0, 0, time.UTC),
		},
	})

	pipe := pipelinePipe{
		input:  input,
		output: output,
	}

	params := PipelineParameters{
		Cfg:        &config.Config{},
		EventsRepo: repo,
	}

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
