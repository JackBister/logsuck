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
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackbister/logsuck/internal/indexedfiles"
	"go.uber.org/dig"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
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

type GlobWatcherCoordinator struct {
	logger       slog.Logger
	watchers     map[string]*GlobWatcher
	indexedFiles []indexedfiles.IndexedFileConfig
	staticConfig config.Config
	configSource config.Source
	publisher    events.Publisher
	ctx          context.Context
}

type GlobWatcherCoordinatorParams struct {
	dig.In

	Logger       *slog.Logger
	StaticConfig *config.Config
	ConfigSource config.Source
	Publisher    events.Publisher
	Ctx          context.Context
}

func NewGlobWatcherCoordinator(p GlobWatcherCoordinatorParams) events.Reader {
	return &GlobWatcherCoordinator{
		logger:       *p.Logger,
		watchers:     map[string]*GlobWatcher{},
		indexedFiles: []indexedfiles.IndexedFileConfig{},
		staticConfig: *p.StaticConfig,
		configSource: p.ConfigSource,
		publisher:    p.Publisher,
		ctx:          p.Ctx,
	}
}

func (gwc *GlobWatcherCoordinator) Start() error {
	r, _ := gwc.configSource.Get()
	indexedFiles, err := indexedfiles.ReadFileConfig(&r.Cfg, &gwc.logger)
	if err != nil {
		gwc.logger.Error("got error when reading dynamic file config", slog.Any("error", err))
		return err
	}
	gwc.indexedFiles = indexedFiles
	gwc.reloadFileWatchers()

	changes := gwc.configSource.Changes()
	for {
		<-changes
		newCfg, err := gwc.configSource.Get()
		if err != nil {
			gwc.logger.Warn("got error when reading updated dynamic file config. file config will not be updated", slog.Any("error", err))
			continue
		}
		newIndexedFiles, err := indexedfiles.ReadFileConfig(&newCfg.Cfg, &gwc.logger)
		if err != nil {
			gwc.logger.Warn("got error when reading updated dynamic file config. file config will not be updated", slog.Any("error", err))
		} else {
			gwc.indexedFiles = newIndexedFiles
			gwc.reloadFileWatchers()
		}
	}
}

func (gwc *GlobWatcherCoordinator) reloadFileWatchers() {
	gwc.logger.Info("reloading file watchers", slog.Int("newIndexedFilesLen", len(gwc.indexedFiles)), slog.Int("oldIndexedFilesLen", len(gwc.watchers)))
	indexedFilesMap := map[string]indexedfiles.IndexedFileConfig{}
	for _, cfg := range gwc.indexedFiles {
		indexedFilesMap[cfg.Filename] = cfg
	}
	watchersToDelete := []string{}
	// Update existing watchers and find watchers to delete
	for k, v := range gwc.watchers {
		newCfg, ok := indexedFilesMap[k]
		if !ok {
			gwc.logger.Info("filename not found in new indexed files config. will cancel and delete watcher", slog.String("fileName", k))
			v.Cancel()
			watchersToDelete = append(watchersToDelete, k)
			continue
		}
		v.UpdateConfig(newCfg)
	}

	// delete watchers that do not exist in the new config
	for _, k := range watchersToDelete {
		delete(gwc.watchers, k)
	}

	// Add new watchers
	for k, v := range indexedFilesMap {
		_, ok := (gwc.watchers)[k]
		if ok {
			continue
		}
		gwc.logger.Info("creating new watcher", slog.String("fileName", k))
		w, err := NewGlobWatcher(v, v.Filename, gwc.staticConfig.HostName, gwc.publisher, gwc.ctx, gwc.logger)
		if err != nil {
			gwc.logger.Warn("got error when creating GlobWatcher", slog.String("fileName", v.Filename), slog.Any("error", err))
			continue
		}
		go w.Start()
		(gwc.watchers)[k] = w.(*GlobWatcher)
	}
}

// GlobWatcher watches a glob pattern to find log files. When a log file is found it will create a FileWatcher to read the file.
type GlobWatcher struct {
	glob string

	fileConfig indexedfiles.IndexedFileConfig
	m          map[string]*FileWatcher
	ctx        context.Context
	Cancel     func()

	hostName string

	eventPublisher events.Publisher

	logger slog.Logger
}

func (gw *GlobWatcher) UpdateConfig(cfg indexedfiles.IndexedFileConfig) {
	gw.logger.Info("updating fileConfig for GlobWatcher",
		slog.String("fileName", gw.fileConfig.Filename))
	if cfg == gw.fileConfig {
		gw.logger.Info("new config for GlobWatcher is the same as before. will not do anything",
			slog.String("fileName", gw.fileConfig.Filename))
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
	eventPublisher events.Publisher
	file           *os.File

	currentSourceId string
	currentOffset   int64
	readBuf         []byte
	workingBuf      []byte

	logger slog.Logger
}

// NewGlobWatcher creates a new watcher. The watcher will find any log files matching the glob pattern and create new FileWatchers for them.
// The FileWatchers will publish events to the given eventPublisher.
func NewGlobWatcher(
	fileConfig indexedfiles.IndexedFileConfig,
	glob string,
	hostName string,
	eventPublisher events.Publisher,
	ctx context.Context,

	logger slog.Logger,
) (events.Reader, error) {
	gwCtx, cancel := context.WithCancel(ctx)
	gw := &GlobWatcher{
		glob: glob,

		fileConfig: fileConfig,
		m:          map[string]*FileWatcher{},
		ctx:        gwCtx,
		Cancel:     cancel,

		hostName: hostName,

		eventPublisher: eventPublisher,

		logger: logger,
	}

	return gw, nil
}

func (gw *GlobWatcher) Start() error {
	absGlob, err := filepath.Abs(gw.glob)
	if err != nil {
		return fmt.Errorf("error geting absGlob for glob=%s: %w", gw.glob, err)
	}
	dir, err := filepath.Abs(filepath.Dir(gw.glob))
	if err != nil {
		return fmt.Errorf("error getting directory for glob=%s: %w", gw.glob, err)
	}
	fsWatchersLock.Lock()
	defer fsWatchersLock.Unlock()
	var watcher *fsnotify.Watcher
	if w, ok := fsWatchers[dir]; ok {
		watcher = w
	} else {
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("error creating FileWatcher for dir=%s, filename=%s: %w", dir, gw.glob, err)
		}
		err = watcher.Add(dir)
		if err != nil {
			return fmt.Errorf("error adding dir to FileWatcher for dir=%s, filename=%s: %w", dir, gw.glob, err)
		}
	}
	initial, err := filepath.Glob(gw.glob)
	if err != nil {
		return fmt.Errorf("got error when globbing using glob=%s: %w", gw.glob, err)
	}
	for _, file := range initial {
		absPath, err := filepath.Abs(file)
		if err != nil {
			gw.logger.Warn("got error when performing filepath.Abs(file)",
				slog.String("dir", dir),
				slog.String("file", file),
				slog.Any("error", err))
			continue
		}
		fw, err := NewFileWatcher(gw.fileConfig, absPath, gw.hostName, gw.eventPublisher, gw.ctx, gw.logger)
		if err != nil {
			gw.logger.Warn("got error when creating new FileWatcher for filename matching glob",
				slog.String("fileName", absPath),
				slog.String("glob", gw.glob),
				slog.Any("error", err))
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
					gw.logger.Warn("got error when matching glob against path",
						slog.String("glob", gw.glob),
						slog.String("path", path),
						slog.Any("error", err))
					continue
				}
				if !matched {
					gw.logger.Info("path does not match glob, skipping",
						slog.String("path", path),
						slog.String("glob", absGlob))
					continue
				}

				absPath, err := filepath.Abs(path)
				if err != nil {
					gw.logger.Warn("got error when performing filepath.Abs(dir/evt.Name) after receiving fsnotify",
						slog.String("dir", dir),
						slog.String("evtName", evt.Name),
						slog.Any("error", err))
					continue
				}

				if fw, ok := gw.m[absPath]; ok {
					fw.commands <- CommandReopen
				} else {
					fw, err = NewFileWatcher(gw.fileConfig, absPath, gw.hostName, gw.eventPublisher, gw.ctx, gw.logger)
					if err != nil {
						gw.logger.Warn("got error when creating new FileWatcher for filename matching glob",
							slog.String("fileName", absPath),
							slog.String("glob", gw.glob),
							slog.Any("error", err))
						continue
					}
					go fw.Start()
					gw.m[absPath] = fw
				}
			}
		}
	}()
	return nil
}

// NewFileWatcher returns a FileWatcher which will watch a file and publish events according to the IndexedFileConfig
func NewFileWatcher(
	fileConfig indexedfiles.IndexedFileConfig,
	filename string,
	hostName string,
	eventPublisher events.Publisher,
	ctx context.Context,
	logger slog.Logger,
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
			slog.String("fileName", fw.filename))
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
					slog.String("fileName", fw.filename))
			} else {
				fw.file = f
				fw.currentSourceId = uuid.NewString()
				fw.currentOffset = 0
				fw.workingBuf = fw.workingBuf[:0]
				fw.logger.Info("opened file",
					slog.String("fileName", fw.filename),
					slog.String("sourceId", fw.currentSourceId))
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
				slog.String("fileName", fw.filename),
				slog.Any("error", err))
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
