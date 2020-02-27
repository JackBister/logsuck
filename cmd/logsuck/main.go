package main

import (
	"flag"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/web"
)

var cfg = config.Config{
	IndexedFiles: []config.IndexedFileConfig{},

	EnableWeb: true,
	HttpAddr:  ":8080",
}

func main() {
	flag.Parse()

	cfg.IndexedFiles = make([]config.IndexedFileConfig, len(flag.Args()))
	for i, file := range flag.Args() {
		cfg.IndexedFiles[i] = config.IndexedFileConfig{
			Filename:       file,
			EventDelimiter: regexp.MustCompile("\n"),
			ReadInterval:   1 * time.Second,
		}
	}

	commandChannels := make([]chan files.FileWatcherCommand, len(cfg.IndexedFiles))
	repo := events.InMemoryRepository()

	for i, file := range cfg.IndexedFiles {
		commandChannels[i] = make(chan files.FileWatcherCommand)
		f, err := os.Open(file.Filename)
		if err != nil {
			log.Fatal(err)
		}
		fw := files.NewFileWatcher(file, commandChannels[i], events.DebugEventPublisher(events.RepositoryEventPublisher(repo)), f)
		go fw.Start()
	}

	log.Fatal(web.NewWeb(cfg, repo).Serve())
}
