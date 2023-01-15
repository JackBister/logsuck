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
	"regexp"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"

	_ "github.com/mattn/go-sqlite3"
)

type TestConfigSource struct {
	config config.Config
}

func newConfigSource() config.ConfigSource {
	return &TestConfigSource{
		config: config.Config{

			SQLite: &config.SqliteConfig{
				DatabaseFile: ":memory:",
				TrueBatch:    true,
			},

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
					Regex: &parser.RegexParserConfig{
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
	repo, err := events.SqliteRepository(db, &config.SqliteConfig{})
	if err != nil {
		t.Fatalf("newInMemRepo got error when creating events repo: %v", err)
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
