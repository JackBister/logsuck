package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const filterStreamPageSize = 1000

type sqliteRepository struct {
	db *sql.DB
}

func SqliteRepository(db *sql.DB) (Repository, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Events (id INTEGER NOT NULL PRIMARY KEY, source TEXT, timestamp DATETIME);")
	if err != nil {
		return nil, fmt.Errorf("error creating events table: %w", err)
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS IX_Events_Timestamp ON Events(timestamp);")
	if err != nil {
		return nil, fmt.Errorf("error creating events timestamp index: %w", err)
	}
	// It seems we have to use FTS4 instead of FTS5? - I could not find an option equivalent to order=DESC for FTS5 and order=DESC makes queries 8-9x faster...
	_, err = db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS EventRaws USING fts4 (raw TEXT, order=DESC);")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws table: %w", err)
	}
	return &sqliteRepository{
		db: db,
	}, nil
}

func (repo *sqliteRepository) Add(evt Event) (*int64, error) {
	startTime := time.Now()
	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	res, err := tx.Exec("INSERT INTO EventRaws (raw) VALUES(?);", evt.Raw)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error executing add raw statement: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error getting event id after insert: %w", err)
	}
	_, err = tx.Exec("INSERT INTO Events(id, source, timestamp) SELECT LAST_INSERT_ROWID(), ?, ?;", evt.Source, evt.Timestamp)
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
		res, err := tx.Exec("INSERT INTO EventRaws (raw) VALUES(?);", evt.Raw)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error executing add raw statement: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error getting event id after insert: %w", err)
		}
		_, err = tx.Exec("INSERT INTO Events(id, source, timestamp) SELECT LAST_INSERT_ROWID(), ?, ?;", evt.Source, evt.Timestamp)
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

func (repo *sqliteRepository) FilterStream(sources, notSources map[string]struct{}, fragments map[string]struct{}, startTime, endTime *time.Time) <-chan []EventWithId {
	ret := make(chan []EventWithId)
	go func() {
		defer close(ret)
		res, err := repo.db.Query("SELECT MAX(id) FROM Events;")
		if err != nil {
			log.Println("error when getting max(id) from Events table in FilterStream:", err)
			return
		}
		if !res.Next() {
			res.Close()
			log.Println("weird state in FilterStream, expected one result when getting max(id) from Events but got 0")
			return
		}
		var maxID int
		err = res.Scan(&maxID)
		res.Close()
		if err != nil {
			log.Println("error when scanning max(id) in FilterStream:", err)
			return
		}
		offset := 0
		for {
			stmt := "SELECT e.id, e.source, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id < " + strconv.Itoa(maxID)
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
			for frag := range fragments {
				stmt += " AND r.raw MATCH '" + frag + "'"
			}
			stmt += " ORDER BY e.timestamp DESC, e.id DESC LIMIT " + strconv.Itoa(filterStreamPageSize) + " OFFSET " + strconv.Itoa(offset) + ";"
			log.Println("executing stmt", stmt)
			res, err = repo.db.Query(stmt)
			if err != nil {
				log.Println("error when getting filtered events in FilterStream:", err)
				return
			}
			evts := make([]EventWithId, 0, filterStreamPageSize)
			eventsInPage := 0
			for res.Next() {
				var evt EventWithId
				err := res.Scan(&evt.Id, &evt.Source, &evt.Timestamp, &evt.Raw)
				if err != nil {
					log.Printf("error when scanning result in FilterStream: %v\n", err)
				} else {
					evts = append(evts, evt)
				}
				eventsInPage++
			}
			res.Close()
			ret <- evts
			if eventsInPage < filterStreamPageSize {
				return
			}
			offset += filterStreamPageSize
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
