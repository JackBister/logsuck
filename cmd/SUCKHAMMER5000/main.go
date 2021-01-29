package main

import (
	"context"
	"database/sql"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logdunkPath := flag.String("logdunk", "./logdunk", "The path to the logdunk executable")
	logsuckPath := flag.String("logsuck", "./logsuck", "The path to the logsuck executable")
	preserveFiles := flag.Bool("preserve", false, "Whether the logsuck.db and generated log file should be preserved after the run")
	runLength := flag.Duration("runlength", 5*time.Second, "The duration to run logdunk for during the benchmark")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	cancelled := false
	defer func() {
		if !cancelled {
			cancel()
		}
		if !*preserveFiles {
			os.Remove("logsuck.db")
			os.Remove("log-0.txt")
		}
	}()
	os.Remove("logsuck.db")
	os.Remove("log-0.txt")
	logdunkCmd := exec.CommandContext(ctx, *logdunkPath, "-numFiles", "1", "-sleepTime", "0ns")
	logsuckCmd := exec.CommandContext(ctx, *logsuckPath, "./log-0.txt")

	// TODO: Change order here when https://github.com/JackBister/logsuck/issues/22 is fixed
	err := logdunkCmd.Start()
	if err != nil {
		log.Fatalf("got error when starting logdunk: %v", err)
		return
	}
	err = logsuckCmd.Start()
	if err != nil {
		log.Fatalf("got error when starting logsuck: %v", err)
		return
	}

	<-time.After(*runLength)

	cancel()
	cancelled = true

	logsuckCmd.Wait()
	logdunkCmd.Wait()

	f, err := os.Open("log-0.txt")
	if err != nil {
		log.Fatalf("failed to open log-0.txt to count generated lines: %v", err)
		return
	}

	generated := 0
	b := make([]byte, 1024*1024)
	for {
		n, err := f.Read(b)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read log-0.txt to count generated lines: %v", err)
			return
		}
		for _, b := range b[:n] {
			if b == '\n' {
				generated++
			}
		}
	}

	db, err := sql.Open("sqlite3", "logsuck.db"+"?cache=shared&_journal_mode=WAL")
	defer db.Close()
	if err != nil {
		log.Fatalf("failed to open logsuck.db to count processed lines: %v", err)
		return
	}

	rows, err := db.Query("SELECT COUNT(1) FROM Events;")
	if err != nil {
		log.Fatalf("failed to query logsuck.db to count processed lines: %v", err)
		return
	}
	rows.Next()
	var processed int
	rows.Scan(&processed)
	rows.Close()

	log.Printf("runLength=%v, generated=%v, processed=%v, perSecond=%v", *runLength, generated, processed, float64(processed)/runLength.Seconds())
}
