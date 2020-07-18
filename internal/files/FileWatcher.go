package files

import (
	"io"
	"log"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
)

// FileWatcherCommand is a command that can be sent to a FileWatcher to tell it to perform various actions
type FileWatcherCommand int

const (
	// CommandStop stops the FileWatcher and cleans up any resources it may have created
	CommandStop FileWatcherCommand = 0
)

// FileWatcher watches files and publishes events as they are written to the file.
type FileWatcher struct {
	fileConfig config.IndexedFileConfig

	commands       <-chan FileWatcherCommand
	eventPublisher events.EventPublisher
	fileReader     io.Reader

	currentOffset int64
	readBuf       []byte
	workingBuf    []byte
}

// NewFileWatcher returns a FileWatcher which will watch a file and publish events according to the IndexedFileConfig
func NewFileWatcher(
	fileConfig config.IndexedFileConfig,
	commands <-chan FileWatcherCommand,
	eventPublisher events.EventPublisher,
	fileReader io.Reader) *FileWatcher {
	return &FileWatcher{
		fileConfig: fileConfig,

		commands:       commands,
		eventPublisher: eventPublisher,
		fileReader:     fileReader,

		currentOffset: 0,
		readBuf:       make([]byte, 4096),
		workingBuf:    make([]byte, 0, 4096),
	}
}

// Start begins watching the file according to its IndexedFileConfig
func (fw *FileWatcher) Start() {
	ticker := time.NewTicker(fw.fileConfig.ReadInterval)
	defer ticker.Stop()
out:
	for {
		select {
		case <-fw.commands:
			break out
		case <-ticker.C: // Proceed
		}
		fw.readToEnd()
	}
}

func (fw *FileWatcher) readToEnd() {
	for read, err := fw.fileReader.Read(fw.readBuf); read != 0; read, err = fw.fileReader.Read(fw.readBuf) {
		if err != nil && err != io.EOF {
			log.Println("Unexpected error=" + err.Error() + ", will abort FileWatcher for filename=" + fw.fileConfig.Filename)
			break
		}
		fw.workingBuf = append(fw.workingBuf, fw.readBuf[:read]...)
		if fw.fileConfig.EventDelimiter.Match(fw.workingBuf) {
			fw.handleEvents()
		}
	}
}

func (fw *FileWatcher) handleEvents() {
	s := string(fw.workingBuf)
	// TODO: Maybe EventDelimiter should just be a string so we don't have to do this
	// Currently the delimiter between each event could in theory have a different length every time
	// so we need to look them up to get the offset right
	delimiters := fw.fileConfig.EventDelimiter.FindAllString(s, -1)
	split := fw.fileConfig.EventDelimiter.Split(s, -1)
	for i, raw := range split[:len(split)-1] {
		evt := events.RawEvent{
			Raw:    raw,
			Source: fw.fileConfig.Filename,
			Offset: fw.currentOffset,
		}
		fw.eventPublisher.PublishEvent(evt, fw.fileConfig.TimeLayout)
		fw.currentOffset += int64(len(raw)) + int64(len(delimiters[i]))
	}
	fw.workingBuf = fw.workingBuf[:0]
	fw.workingBuf = append(fw.workingBuf, []byte(split[len(split)-1])...)
}
