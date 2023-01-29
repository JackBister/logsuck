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

package events

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"go.uber.org/zap"

	_ "github.com/mattn/go-sqlite3"
)

func TestAddBatchTrueBatch(t *testing.T) {
	repo := createRepoWithCfg(t, &config.SqliteConfig{
		DatabaseFile: ":memory:",
		TrueBatch:    true,
	})

	repo.AddBatch([]Event{
		{
			Raw:       "2021-02-01 00:00:00 log event",
			Timestamp: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})

	evts, err := repo.GetByIds([]int64{1}, SortModeNone)
	if err != nil {
		t.Fatalf("got error when retrieving event: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("got unexpected number of events, expected 1 event but got %v", len(evts))
	}
}

func TestAddBatchOneByOne(t *testing.T) {
	repo := createRepoWithCfg(t, &config.SqliteConfig{
		DatabaseFile: ":memory:",
		TrueBatch:    false,
	})

	repo.AddBatch([]Event{
		{
			Raw:       "2021-02-01 00:00:00 log event",
			Timestamp: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})

	evts, err := repo.GetByIds([]int64{1}, SortModeNone)
	if err != nil {
		t.Fatalf("got error when retrieving event: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("got unexpected number of events, expected 1 event but got %v", len(evts))
	}
}

func TestDeleteEmptyList(t *testing.T) {
	repo := createRepo(t)

	repo.AddBatch([]Event{
		{
			Raw:       "2022-01-27 00:00:00 my event",
			Timestamp: time.Date(2022, 1, 27, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})

	err := repo.DeleteBatch([]int64{})
	if err != nil {
		t.Fatalf("got error when deleting empty list of eventIds: %v", err)
	}
	evts, err := repo.GetByIds([]int64{1}, SortModeNone)
	if err != nil {
		t.Fatalf("got error when getting events after deleting empty list: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 event in repository after deleting empty list but got %v", len(evts))
	}
}

func TestDeleteOneEvent(t *testing.T) {
	repo := createRepo(t)

	repo.AddBatch([]Event{
		{
			Raw:       "2022-01-27 00:00:00 my event",
			Timestamp: time.Date(2022, 1, 27, 0, 0, 0, 0, time.UTC),
			Host:      "localhost",
			Source:    "log.txt",
			Offset:    0,
		},
	})
	evts, err := repo.GetByIds([]int64{1}, SortModeNone)
	if err != nil {
		t.Fatalf("got error when getting events after deleting empty list: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 event in repository before deleting it but got %v", len(evts))
	}
	err = repo.DeleteBatch([]int64{1})
	if err != nil {
		t.Fatalf("got error when deleting eventId: %v", err)
	}
	evts, err = repo.GetByIds([]int64{1}, SortModeNone)
	if err != nil {
		t.Fatalf("got error when getting events after deleting empty list: %v", err)
	}
	if len(evts) != 0 {
		t.Fatalf("expected 0 events in repository after deleting event but got %v", len(evts))
	}
}

func createRepo(t *testing.T) Repository {
	return createRepoWithCfg(t, &config.SqliteConfig{
		DatabaseFile: ":memory:",
		TrueBatch:    true,
	})
}

func createRepoWithCfg(t *testing.T, cfg *config.SqliteConfig) Repository {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("got error when creating in-memory SQLite database: %v", err)
	}
	repo, err := SqliteRepository(db, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("got error when creating events repo: %v", err)
	}
	return repo
}
