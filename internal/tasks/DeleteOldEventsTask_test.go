// Copyright 2022 Jack Bister
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

package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

func TestDeleteOldEventsTaskInvalidMinAgeDoesNotDelete(t *testing.T) {
	repo := createRepo(t)
	task := &DeleteOldEventsTask{
		Repo: repo,
	}

	repo.AddBatch([]events.Event{
		{
			Raw:       "2022-01-27 00:00:00 my event",
			Timestamp: time.Date(2022, 1, 27, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})

	cfgNullMinAge := map[string]any{}
	task.Run(cfgNullMinAge, context.Background())
	checkForEvent(t, repo)

	cfgEmptyMinAge := map[string]any{
		"minAge": "",
	}
	task.Run(cfgEmptyMinAge, context.Background())
	checkForEvent(t, repo)

	cfgUnparseableMinAge := map[string]any{
		"minAge": "123x",
	}
	task.Run(cfgUnparseableMinAge, context.Background())
	checkForEvent(t, repo)
}

func TestDeleteOldEventsDeletesOldEvents(t *testing.T) {
	repo := createRepo(t)
	task := &DeleteOldEventsTask{
		Repo: repo,
		Now: func() time.Time {
			return time.Date(2022, 1, 27, 20, 0, 0, 0, time.UTC)
		}}

	repo.AddBatch([]events.Event{
		{
			Raw:       "2022-01-27 00:00:00 my event",
			Timestamp: time.Date(2022, 1, 27, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
		{
			Raw:       "2020-01-27 00:00:00 my old event",
			Timestamp: time.Date(2022, 1, 27, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})

	cfg := map[string]any{
		"minAge": "7d",
	}

	task.Run(cfg, context.Background())
}

func checkForEvent(t *testing.T, repo events.Repository) {
	evts, err := repo.GetByIds([]int64{1}, events.SortModeNone)
	if err != nil {
		t.Fatalf("got error when getting events after running task: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected to have 1 event in repository after running task with empty minAge but have %v", len(evts))
	}
}

func createRepo(t *testing.T) events.Repository {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("got error when creating in-memory SQLite database: %v", err)
	}
	repo, err := events.SqliteRepository(db, &config.SqliteConfig{
		DatabaseFile: ":memory:",
		TrueBatch:    true,
	},
		zap.NewNop())
	if err != nil {
		t.Fatalf("got error when creating events repo: %v", err)
	}
	return repo
}
