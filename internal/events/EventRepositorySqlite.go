package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteRepository struct {
	db *sql.DB

	addRawStmt *sql.Stmt
	addStmt    *sql.Stmt
}

func SqliteRepository(fileName string) (*sqliteRepository, error) {
	db, err := sql.Open("sqlite3", fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite database: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Events (id INTEGER NOT NULL PRIMARY KEY, source TEXT, timestamp DATETIME);")
	if err != nil {
		return nil, fmt.Errorf("error creating events table: %w", err)
	}
	_, err = db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS EventRaws USING fts3 (raw TEXT);")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws table: %w", err)
	}
	addRawStmt, err := db.Prepare("INSERT INTO EventRaws (raw) VALUES(?);")
	if err != nil {
		return nil, fmt.Errorf("error preparing add raw statement: %w", err)
	}
	addStmt, err := db.Prepare("INSERT INTO Events(id, source, timestamp) SELECT LAST_INSERT_ROWID(), ?, ?;")
	if err != nil {
		return nil, fmt.Errorf("error preparing add statement: %w", err)
	}
	return &sqliteRepository{
		db:         db,
		addRawStmt: addRawStmt,
		addStmt:    addStmt,
	}, nil
}

func (repo *sqliteRepository) Add(evt Event) (*int64, error) {
	startTime := time.Now()
	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	res, err := tx.Stmt(repo.addRawStmt).Exec(evt.Raw)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error executing add raw statement: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error getting event id after insert: %w", err)
	}
	_, err = tx.Stmt(repo.addStmt).Exec(evt.Source, evt.Timestamp)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error executing add statement: %w", err)
	}
	err = tx.Commit()
	if err != nil {
		// TODO: Hmm?
	}
	log.Printf("added event in timeInMs=%v\n", time.Now().Sub(startTime).Milliseconds())
	return &id, nil
}

func (repo *sqliteRepository) AddBatch(events []Event) ([]int64, error) {
	startTime := time.Now()
	ret := make([]int64, len(events))
	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	for i, evt := range events {
		res, err := tx.Stmt(repo.addRawStmt).Exec(evt.Raw)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error executing add raw statement: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error getting event id after insert: %w", err)
		}
		_, err = tx.Stmt(repo.addStmt).Exec(evt.Source, evt.Timestamp)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error executing add statement: %w", err)
		}
		ret[i] = id
	}
	err = tx.Commit()
	if err != nil {
		// TODO: Hmm?
	}
	log.Printf("added numEvents=%v in timeInMs=%v\n", len(events), time.Now().Sub(startTime).Milliseconds())
	return ret, nil
}

func (repo *sqliteRepository) FilterStream(sources, notSources map[string]struct{}, startTime, endTime *time.Time) <-chan EventWithId {
	ret := make(chan EventWithId)
	go func() {
		defer close(ret)
		stmt := "SELECT e.id, e.source, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE 1=1"
		if startTime != nil {
			stmt += " AND e.timestamp >= '" + startTime.String() + "'"
		}
		if endTime != nil {
			stmt += " AND e.timestamp <= '" + endTime.String() + "'"
		}
		for s := range sources {
			compiledSource := strings.Replace(s, "*", "%", -1)
			stmt += " AND e.source LIKE '" + compiledSource + "'"
		}
		for s := range notSources {
			compiledSource := strings.Replace(s, "*", "%", -1)
			stmt += " AND e.source NOT LIKE '" + compiledSource + "'"
		}
		stmt += " ORDER BY e.timestamp DESC"
		log.Println("executing stmt", stmt)
		res, err := repo.db.Query(stmt)
		if err != nil {
			return
		}
		for res.Next() {
			var evt EventWithId
			err := res.Scan(&evt.Id, &evt.Source, &evt.Timestamp, &evt.Raw)
			if err != nil {
				log.Printf("error when scanning result in FilterStream: %v\n", err)
			} else {
				ret <- evt
			}
		}
	}()
	return ret
}

func (repo *sqliteRepository) GetByIds(ids []int64) ([]EventWithId, error) {
	ret := make([]EventWithId, len(ids))

	// TODO: I'm PRETTY sure this code is garbage
	stmt := "SELECT e.id, e.source, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id IN ("
	for i, id := range ids {
		if i == len(ids)-1 {
			stmt += strconv.FormatInt(id, 10)
		} else {
			stmt += strconv.FormatInt(id, 10) + ","
		}
	}
	stmt += ");"

	res, err := repo.db.Query(stmt)
	if err != nil {
		return nil, fmt.Errorf("error executing GetByIds query: %w", err)
	}
	defer res.Close()

	idx := 0
	for res.Next() {
		err = res.Scan(&ret[idx].Id, &ret[idx].Source, &ret[idx].Timestamp, &ret[idx].Raw)
		if err != nil {
			return nil, fmt.Errorf("error when scanning row in GetByIds: %w", err)
		}
		idx++
	}

	return ret, nil
}
