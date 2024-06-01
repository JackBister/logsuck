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
	"database/sql"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"

	"github.com/jackbister/logsuck/plugins/sqlite_events"
)

type TestConfigSource struct {
	config config.Config
}

func newConfigSource() config.Source {
	return &TestConfigSource{
		config: config.Config{
			Files: map[string]config.FileConfig{
				"my-log.txt": {
					Filename: "my-log.txt",
				},
			},

			FileTypes: map[string]config.FileTypeConfig{
				"DEFAULT": {
					Name:         "DEFAULT",
					TimeLayout:   "2006/01/02 15:04:05",
					ReadInterval: 1 * time.Second,
					ParserType:   config.ParserTypeRegex,
					Regex: &config.RegexParserConfig{
						EventDelimiter: regexp.MustCompile("\n"),
						FieldExtractors: []*regexp.Regexp{
							regexp.MustCompile("(\\w+)=(\\w+)"),
							regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d\\.\\d\\d\\d\\d\\d\\d)"),
						},
					},
				},
			},

			HostTypes: map[string]config.HostTypeConfig{
				"DEFAULT": {
					Files: []config.HostFileConfig{
						{
							Name: "my-log.txt",
						},
					},
				},
			},
		},
	}
}

func (s *TestConfigSource) Get() (*config.ConfigResponse, error) {
	return &config.ConfigResponse{Modified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Cfg: s.config}, nil
}

func (s *TestConfigSource) Changes() <-chan struct{} {
	return make(<-chan struct{})
}

func newInMemRepo(t *testing.T) events.Repository {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("newInMemRepo got error when creating in-memory SQLite database: %v", err)
	}
	repo, err := sqlite_events.NewSqliteEventRepository(sqlite_events.SqliteEventRepositoryParams{
		Db: db,
		Cfg: &sqlite_events.Config{
			TrueBatch: true,
		},
		Logger: slog.Default(),
	})
	if err != nil {
		t.Fatalf("newInMemRepo got error when creating events repo: %v", err)
	}
	return repo
}

func newPipe() (pipe pipeline.Pipe, in, out chan pipeline.StepResult) {
	in = make(chan pipeline.StepResult)
	out = make(chan pipeline.StepResult)

	pipe = pipeline.Pipe{
		Input:  in,
		Output: out,
	}

	return pipe, in, out
}
