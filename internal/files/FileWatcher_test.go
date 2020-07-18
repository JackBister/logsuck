package files

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
)

var TestConfig = config.IndexedFileConfig{
	Filename:       "testlog",
	EventDelimiter: regexp.MustCompile("\n"),
	ReadInterval:   time.Millisecond,
	TimeLayout:     "2006/01/02 15:04:05",
}

type testEventPublisher struct {
	events []events.RawEvent
}

func (ep *testEventPublisher) PublishEvent(evt events.RawEvent, _ string) {
	ep.events = append(ep.events, evt)
}

func TestFileWatcher_PublishesInitialEvent(t *testing.T) {
	const LogContent = "Initial log content"
	eventPublisher := testEventPublisher{events: make([]events.RawEvent, 0)}
	commandChannel := make(chan FileWatcherCommand)
	buffer := bytes.NewBufferString(LogContent + "\n")
	fw := NewFileWatcher(TestConfig, commandChannel, &eventPublisher, buffer)

	fw.readToEnd()

	if len(eventPublisher.events) != 1 {
		t.Error("FileWatcher did not publish an event with the initial log content")
	}

	evt := eventPublisher.events[0]
	if evt.Raw != LogContent {
		t.Error("Published event did not contain the correct raw string, expected=", LogContent, "got=", evt.Raw)
	}

	if evt.Source != TestConfig.Filename {
		t.Error("Published event's Source does not match, expected=", TestConfig.Filename, "got=", evt.Source)
	}
}

func TestFileWatcher_PublishesLaterEvent(t *testing.T) {
	const AddedLogContent = "Added log content"
	eventPublisher := testEventPublisher{events: make([]events.RawEvent, 0)}
	commandChannel := make(chan FileWatcherCommand)
	buffer := bytes.NewBufferString("Initial log content\n")
	fw := NewFileWatcher(TestConfig, commandChannel, &eventPublisher, buffer)

	fw.readToEnd()
	buffer.WriteString(AddedLogContent + "\n")
	fw.readToEnd()

	if len(eventPublisher.events) != 2 {
		t.Error("Expected 2 events to be published but got", len(eventPublisher.events))
	}

	evt := eventPublisher.events[1]
	if evt.Raw != AddedLogContent {
		t.Error("Published event did not contain the correct raw string, expected=", AddedLogContent, "got=", evt.Raw)
	}

	if evt.Source != TestConfig.Filename {
		t.Error("Published event's Source does not match, expected=", TestConfig.Filename, "got=", evt.Source)
	}

}

func TestFileWatcher_StopsWhenAsked(t *testing.T) {
	commandChannel := make(chan FileWatcherCommand)
	buffer := bytes.NewBufferString("Initial log content\n")

	fw := NewFileWatcher(TestConfig, commandChannel, events.NopEventPublisher(), buffer)
	go fw.Start()

	commandChannel <- CommandStop
	// If the watcher is broken, this test will time out
}
