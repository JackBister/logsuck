package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/jackbister/logsuck/internal/jobs"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/files"
	"github.com/jackbister/logsuck/internal/web"

	_ "github.com/mattn/go-sqlite3"
)

var cfg = config.Config{
	IndexedFiles: []config.IndexedFileConfig{},

	FieldExtractors: []*regexp.Regexp{
		regexp.MustCompile("(\\w+)=(\\w+)"),
		regexp.MustCompile("^(?P<_time>\\d\\d\\d\\d/\\d\\d/\\d\\d \\d\\d:\\d\\d:\\d\\d.\\d\\d\\d\\d\\d\\d)"),
	},
	TimeLayout: "2006/01/02 15:04:05",

	DatabaseFile: "logsuck.db",

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
	db, err := sql.Open("sqlite3", cfg.DatabaseFile+"?cache=shared")
	if err != nil {
		log.Fatalln(err.Error())
	}
	db.SetMaxOpenConns(1)
	repo, err := events.SqliteRepository(db)
	if err != nil {
		log.Fatalln(err.Error())
	}
	publisher := events.BatchedRepositoryPublisher(&cfg, repo)
	jobRepo, err := jobs.SqliteRepository(db)
	if err != nil {
		log.Fatalln(err.Error())
	}

	for i, file := range cfg.IndexedFiles {
		commandChannels[i] = make(chan files.FileWatcherCommand)
		f, err := os.Open(file.Filename)
		if err != nil {
			log.Fatal(err)
		}
		fw := files.NewFileWatcher(file, commandChannels[i], publisher, f)
		log.Println("Starting FileWatcher for filename=" + file.Filename)
		go fw.Start()
	}

	log.Fatal(web.NewWeb(&cfg, repo, jobRepo).Serve())
}
