package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"

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

	cfgNullMinAge := config.NewDynamicConfig([]config.ConfigSource{
		config.NewMapConfigSource(map[string]string{}),
	})
	task.Run(cfgNullMinAge, context.Background())
	checkForEvent(t, repo)

	cfgEmptyMinAge := config.NewDynamicConfig([]config.ConfigSource{
		config.NewMapConfigSource(map[string]string{
			"minAge": "",
		}),
	})
	task.Run(cfgEmptyMinAge, context.Background())
	checkForEvent(t, repo)

	cfgUnparseableMinAge := config.NewDynamicConfig([]config.ConfigSource{
		config.NewMapConfigSource(map[string]string{
			"minAge": "123x",
		}),
	})
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

	cfg := config.NewDynamicConfig([]config.ConfigSource{
		config.NewMapConfigSource(map[string]string{
			"minAge": "7d",
		}),
	})

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
	})
	if err != nil {
		t.Fatalf("got error when creating events repo: %v", err)
	}
	return repo
}
