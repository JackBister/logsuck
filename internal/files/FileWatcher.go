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

package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/indexedfiles"
	"go.uber.org/zap"

	"github.com/fsnotify/fsnotify"
)

// FileWatcherCommand is a command that can be sent to a FileWatcher to tell it to perform various actions
type FileWatcherCommand int

const (
	// CommandReopen closes the file and opens it again
	CommandReopen FileWatcherCommand = 1
	// CommandReload updates the file watcher's configuration to the new configuration stored in the "newFileConfig" property
	CommandReloadConfig FileWatcherCommand = 2
)

// There is probably a cleaner solution to this.
// Maybe we could just have one fsnotify.Watcher for all files since we check for glob match anyway?
var fsWatchers = map[string]*fsnotify.Watcher{}
var fsWatchersLock = sync.Mutex{}

// GlobWatcher watches a glob pattern to find log files. When a log file is found it will create a FileWatcher to read the file.
type GlobWatcher struct {
	fileConfig indexedfiles.IndexedFileConfig
	m          map[string]*FileWatcher
	ctx        context.Context

	Cancel func()

	logger *zap.Logger
}

func (gw *GlobWatcher) UpdateConfig(cfg indexedfiles.IndexedFileConfig) {
	gw.logger.Info("updating fileConfig for GlobWatcher",
		zap.String("fileName", gw.fileConfig.Filename))
	if cfg == gw.fileConfig {
		gw.logger.Info("new config for GlobWatcher is the same as before. will not do anything",
			zap.String("fileName", gw.fileConfig.Filename))
		return
	}
	for _, v := range gw.m {
		v.newFileConfig = &cfg
		v.commands <- CommandReloadConfig
	}
}

// FileWatcher watches files and publishes events as they are written to the file.
type FileWatcher struct {
	fileConfig    indexedfiles.IndexedFileConfig
	newFileConfig *indexedfiles.IndexedFileConfig
	ctx           context.Context

	filename string
	hostName string

	commands       chan FileWatcherCommand
	eventPublisher events.EventPublisher
	file           *os.File

	currentSourceId string
	currentOffset   int64
	readBuf         []byte
	workingBuf      []byte

	logger *zap.Logger
}

// NewGlobWatcher creates a new watcher. The watcher will find any log files matching the glob pattern and create new FileWatchers for them.
// The FileWatchers will publish events to the given eventPublisher.
func NewGlobWatcher(
	fileConfig indexedfiles.IndexedFileConfig,
	glob string,
	hostName string,
	eventPublisher events.EventPublisher,
	ctx context.Context,

	logger *zap.Logger,
) (*GlobWatcher, error) {
	absGlob, err := filepath.Abs(glob)
	if err != nil {
		return nil, fmt.Errorf("error geting absGlob for glob=%s: %w", glob, err)
	}
	dir, err := filepath.Abs(filepath.Dir(glob))
	if err != nil {
		return nil, fmt.Errorf("error getting directory for glob=%s: %w", glob, err)
	}
	fsWatchersLock.Lock()
	defer fsWatchersLock.Unlock()
	var watcher *fsnotify.Watcher
	if w, ok := fsWatchers[dir]; ok {
		watcher = w
	} else {
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("error creating FileWatcher for dir=%s, filename=%s: %w", dir, glob, err)
		}
		err = watcher.Add(dir)
		if err != nil {
			return nil, fmt.Errorf("error adding dir to FileWatcher for dir=%s, filename=%s: %w", dir, glob, err)
		}
	}

	gwCtx, cancel := context.WithCancel(ctx)

	gw := &GlobWatcher{
		fileConfig: fileConfig,
		m:          map[string]*FileWatcher{},
		ctx:        gwCtx,
		Cancel:     cancel,
	}

	initial, err := filepath.Glob(glob)
	if err != nil {
		return nil, fmt.Errorf("got error when globbing using glob=%s: %w", glob, err)
	}
	for _, file := range initial {
		absPath, err := filepath.Abs(file)
		if err != nil {
			logger.Warn("got error when performing filepath.Abs(file)",
				zap.String("dir", dir),
				zap.String("file", file),
				zap.Error(err))
			continue
		}
		fw, err := NewFileWatcher(fileConfig, absPath, hostName, eventPublisher, gwCtx, logger.Named("FileWatcher"))
		if err != nil {
			logger.Warn("got error when creating new FileWatcher for filename matching glob",
				zap.String("fileName", absPath),
				zap.String("glob", glob),
				zap.Error(err))
			continue
		}
		go fw.Start()
		gw.m[absPath] = fw
	}

	go func() {
		for {
			select {
			case <-gw.ctx.Done():
				return
			case evt := <-watcher.Events:
				if evt.Op&(fsnotify.Create|fsnotify.Remove) == 0 {
					continue
				}
				path := evt.Name
				matched, err := filepath.Match(absGlob, path)
				if err != nil {
					logger.Warn("got error when matching glob against path",
						zap.String("glob", glob),
						zap.String("path", path),
						zap.Error(err))
					continue
				}
				if !matched {
					logger.Info("path does not match glob, skipping",
						zap.String("path", path),
						zap.String("glob", absGlob))
					continue
				}

				absPath, err := filepath.Abs(path)
				if err != nil {
					logger.Warn("got error when performing filepath.Abs(dir/evt.Name) after receiving fsnotify",
						zap.String("dir", dir),
						zap.String("evtName", evt.Name),
						zap.Error(err))
					continue
				}

				if fw, ok := gw.m[absPath]; ok {
					fw.commands <- CommandReopen
				} else {
					fw, err = NewFileWatcher(fileConfig, absPath, hostName, eventPublisher, gwCtx, logger.Named("FileWatcher"))
					if err != nil {
						logger.Warn("got error when creating new FileWatcher for filename matching glob",
							zap.String("fileName", absPath),
							zap.String("glob", glob),
							zap.Error(err))
						continue
					}
					go fw.Start()
					gw.m[absPath] = fw
				}
			}
		}
	}()

	return gw, nil
}

// NewFileWatcher returns a FileWatcher which will watch a file and publish events according to the IndexedFileConfig
func NewFileWatcher(
	fileConfig indexedfiles.IndexedFileConfig,
	filename string,
	hostName string,
	eventPublisher events.EventPublisher,
	ctx context.Context,
	logger *zap.Logger,
) (*FileWatcher, error) {
	return &FileWatcher{
		fileConfig: fileConfig,
		ctx:        ctx,

		filename: filename,
		hostName: hostName,

		commands:       make(chan FileWatcherCommand),
		eventPublisher: eventPublisher,
		file:           nil,

		currentOffset: 0,
		readBuf:       make([]byte, 4096),
		workingBuf:    make([]byte, 0, 4096),

		logger: logger,
	}, nil
}

// Start begins watching the file according to its IndexedFileConfig
func (fw *FileWatcher) Start() {
	if fw.fileConfig.FileParser == nil {
		fw.logger.Warn("FileParser is nil. will not watch this file. review your configuration to make sure that this file has an associated file type with a parser configured.",
			zap.String("fileName", fw.filename))
		return
	}
	ticker := time.NewTicker(fw.fileConfig.ReadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-fw.ctx.Done():
			return
		case cmd := <-fw.commands:
			if cmd == CommandReopen && fw.file != nil {
				fw.readToEnd()
				fw.file.Close()
				fw.file = nil
			}
		case <-ticker.C: // Proceed
		}
		if fw.file == nil {
			f, err := os.Open(fw.filename)
			if err != nil {
				fw.logger.Warn("error opening file, will retry later",
					zap.String("fileName", fw.filename))
			} else {
				fw.file = f
				fw.currentSourceId = uuid.NewString()
				fw.currentOffset = 0
				fw.workingBuf = fw.workingBuf[:0]
				fw.logger.Info("opened file",
					zap.String("fileName", fw.filename),
					zap.String("sourceId", fw.currentSourceId))
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
			fw.logger.Error("Unexpected error, will abort FileWatcher",
				zap.String("fileName", fw.filename),
				zap.Error(err))
			break
		}
		fw.workingBuf = append(fw.workingBuf, fw.readBuf[:read]...)
		if fw.fileConfig.FileParser.CanSplit(fw.workingBuf) {
			fw.handleEvents()
		}
	}
}

func (fw *FileWatcher) handleEvents() {
	s := string(fw.workingBuf)
	// TODO: Maybe EventDelimiter should just be a string so we don't have to do this
	// Currently the delimiter between each event could in theory have a different length every time
	// so we need to look them up to get the offset right
	splitResult := fw.fileConfig.FileParser.Split(s)
	for _, res := range splitResult.Events {
		evt := events.RawEvent{
			Raw:      res.Raw,
			Host:     fw.hostName,
			Source:   fw.filename,
			SourceId: fw.currentSourceId,
			Offset:   fw.currentOffset,
		}
		fw.eventPublisher.PublishEvent(evt, fw.fileConfig.TimeLayout, fw.fileConfig.FileParser)
		fw.currentOffset += res.Offset
	}
	fw.workingBuf = fw.workingBuf[:0]
	fw.workingBuf = append(fw.workingBuf, []byte(splitResult.Remainder)...)
}
