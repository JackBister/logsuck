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
	"database/sql"
	"testing"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"

	_ "github.com/mattn/go-sqlite3"
)

func newDynamicConfig() (map[string]interface{}, config.DynamicConfig) {
	m := map[string]interface{}{}
	dc := config.NewDynamicConfig([]config.ConfigSource{
		config.NewMapConfigSource(m),
	})
	return m, dc
}

func newInMemRepo(t *testing.T) events.Repository {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("TestRexPipelineStep got error when creating in-memory SQLite database: %v", err)
	}
	repo, err := events.SqliteRepository(db, &config.SqliteConfig{})
	if err != nil {
		t.Fatalf("TestRexPipelineStep got error when creating events repo: %v", err)
	}
	return repo
}

func newPipe() (pipe pipelinePipe, in, out chan PipelineStepResult) {
	in = make(chan PipelineStepResult)
	out = make(chan PipelineStepResult)

	pipe = pipelinePipe{
		input:  in,
		output: out,
	}

	return pipe, in, out
}
