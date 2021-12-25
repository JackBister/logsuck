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

package files

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"

	"github.com/fsnotify/fsnotify"
)

// FileWatcherCommand is a command that can be sent to a FileWatcher to tell it to perform various actions
type FileWatcherCommand int

const (
	// CommandStop stops the FileWatcher and cleans up any resources it may have created
	CommandStop FileWatcherCommand = 0
	// CommandReopen closes the file and opens it again
	CommandReopen FileWatcherCommand = 1
)

// FileWatcher watches files and publishes events as they are written to the file.
type FileWatcher struct {
	fileConfig config.IndexedFileConfig

	filename string
	hostName string

	commands       chan FileWatcherCommand
	eventPublisher events.EventPublisher
	file           *os.File

	currentSourceId string
	currentOffset   int64
	readBuf         []byte
	workingBuf      []byte
}

// NewFileWatcher returns a FileWatcher which will watch a file and publish events according to the IndexedFileConfig
func NewFileWatcher(
	fileConfig config.IndexedFileConfig,
	filename string,
	hostName string,
	commands chan FileWatcherCommand,
	eventPublisher events.EventPublisher,
) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating FileWatcher for fileName=%s: %w", filename, err)
	}
	err = watcher.Add(filename)
	if err != nil {
		return nil, fmt.Errorf("error creating FileWatcher for fileName=%s: %w", filename, err)
	}
	go func() {
		for {
			evt := <-watcher.Events
			log.Println("received fsnotify", evt)
			// The reasoning for reopening on "Write" is that os.Create actually does not trigger a Remove or Create event if it truncates a file.
			// This may end up being a problem though.
			// TODO: Maybe the write case should be specially handled and just reset the offset/seek position to 0?
			if evt.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove) != 0 {
				log.Printf("filename=%s appears to have been rolled, will try to reopen\n", filename)
				commands <- CommandReopen
			}
		}
	}()
	return &FileWatcher{
		fileConfig: fileConfig,

		filename: filename,
		hostName: hostName,

		commands:       commands,
		eventPublisher: eventPublisher,
		file:           nil,

		currentOffset: 0,
		readBuf:       make([]byte, 4096),
		workingBuf:    make([]byte, 0, 4096),
	}, nil
}

// Start begins watching the file according to its IndexedFileConfig
func (fw *FileWatcher) Start() {
	ticker := time.NewTicker(fw.fileConfig.ReadInterval)
	defer ticker.Stop()
out:
	for {
		select {
		case cmd := <-fw.commands:
			if cmd == CommandStop {
				break out
			} else if cmd == CommandReopen && fw.file != nil {
				fw.readToEnd()
				fw.file.Close()
				fw.file = nil
			}
		case <-ticker.C: // Proceed
		}
		if fw.file == nil {
			f, err := os.Open(fw.filename)
			if err != nil {
				log.Printf("error opening filename=%s, will retry later.\n", fw.filename)
			} else {
				fw.file = f
				fw.currentSourceId = uuid.NewString()
				fw.currentOffset = 0
				fw.workingBuf = fw.workingBuf[:0]
				log.Printf("opened filename=%s with sourceId=%s\n", fw.filename, fw.currentSourceId)
			}
		}
		if fw.file != nil {
			fw.readToEnd()
		}
	}
}

func (fw *FileWatcher) readToEnd() {
	for read, err := fw.file.Read(fw.readBuf); read != 0; read, err = fw.file.Read(fw.readBuf) {
		if err != nil && err != io.EOF {
			log.Println("Unexpected error=" + err.Error() + ", will abort FileWatcher for filename=" + fw.filename)
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
			Raw:      raw,
			Host:     fw.hostName,
			Source:   fw.filename,
			SourceId: fw.currentSourceId,
			Offset:   fw.currentOffset,
		}
		fw.eventPublisher.PublishEvent(evt, fw.fileConfig.TimeLayout)
		fw.currentOffset += int64(len(raw)) + int64(len(delimiters[i]))
	}
	fw.workingBuf = fw.workingBuf[:0]
	fw.workingBuf = append(fw.workingBuf, []byte(split[len(split)-1])...)
}
