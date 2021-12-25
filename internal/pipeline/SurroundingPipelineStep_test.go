package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
)

func TestSurroundingPipelineStep(t *testing.T) {
	sps, err := compileSurroundingStep("", map[string]string{
		"eventId": "3",
	})
	if err != nil {
		t.Fatalf("TestSurroundingPipelineStep got unexpected error: %v", err)
	}
	repo := newInMemRepo(t)
	params := PipelineParameters{
		Cfg:        &config.Config{},
		EventsRepo: repo,
	}
	pipe, input, output := newPipe()
	close(input)
	repo.AddBatch([]events.Event{
		{
			Raw:       "2021-01-20 20:29:00 This is event 1",
			Host:      "MYHOST",
			Offset:    0,
			Source:    "my-log.txt",
			SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
			Timestamp: time.Date(2021, 1, 20, 20, 29, 0, 0, time.UTC),
		},
		{
			Raw:       "2021-02-22 20:29:00 This is an event in my-log-2",
			Host:      "MYHOST",
			Offset:    0,
			Source:    "my-log-2.txt",
			SourceId:  "04b82f34-b4fa-47bb-bae7-2b8771c7dc94",
			Timestamp: time.Date(2021, 2, 22, 20, 29, 0, 0, time.UTC),
		},
		{
			Raw:       "2021-01-20 20:29:01 This is event 2",
			Host:      "MYHOST",
			Offset:    50,
			Source:    "my-log.txt",
			SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
			Timestamp: time.Date(2021, 1, 20, 20, 29, 1, 0, time.UTC),
		},
		{
			Raw:       "2021-01-20 20:29:02 This is event 3",
			Host:      "MYHOST",
			Offset:    100,
			Source:    "my-log.txt",
			SourceId:  "1a9a7cd6-0f00-4aa6-ae2e-1ad17d40bb35",
			Timestamp: time.Date(2021, 1, 20, 20, 29, 2, 0, time.UTC),
		},
		{
			Raw:       "2021-02-22 20:29:01 This is an event in my-log-2",
			Host:      "MYHOST",
			Offset:    50,
			Source:    "my-log-2.txt",
			SourceId:  "04b82f34-b4fa-47bb-bae7-2b8771c7dc94",
			Timestamp: time.Date(2021, 2, 22, 20, 29, 1, 0, time.UTC),
		},
	})

	go sps.Execute(context.Background(), pipe, params)

	result, ok := <-output
	if !ok {
		t.Fatal("TestSurroundingPipelineStep got unexpected !ok when receiving output")
	}
	if len(result.Events) != 3 {
		t.Fatalf("TestSurroundingPipelineStep got unexpected number of events, expected 3 but got %v", len(result.Events))
	}
	if result.Events[0].Raw != "2021-01-20 20:29:02 This is event 3" {
		t.Fatalf("First event did not have the correct raw field, got '%s'", result.Events[2].Raw)
	}
	if result.Events[1].Raw != "2021-01-20 20:29:01 This is event 2" {
		t.Fatalf("Second event did not have the correct raw field, got '%s'", result.Events[1].Raw)
	}
	if result.Events[2].Raw != "2021-01-20 20:29:00 This is event 1" {
		t.Fatalf("First event did not have the correct raw field, got '%s'", result.Events[0].Raw)
	}
	_, ok = <-output
	if ok {
		t.Fatal("TestSurroundingPipelineStep got unexpected ok when receiving output, expected the channel to be closed by now")
	}
}