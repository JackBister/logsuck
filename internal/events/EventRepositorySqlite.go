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

package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/internal/search"
)

const expectedConstraintViolationForDuplicates = "UNIQUE constraint failed: Events.host, Events.source, Events.timestamp, Events.offset"
const filterStreamPageSize = 1000

type sqliteRepository struct {
	db *sql.DB
}

func SqliteRepository(db *sql.DB) (Repository, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Events (id INTEGER NOT NULL PRIMARY KEY, host TEXT NOT NULL, source TEXT NOT NULL, timestamp DATETIME NOT NULL, offset BIGINT NOT NULL, UNIQUE(host, source, timestamp, offset));")
	if err != nil {
		return nil, fmt.Errorf("error creating events table: %w", err)
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS IX_Events_Timestamp ON Events(timestamp);")
	if err != nil {
		return nil, fmt.Errorf("error creating events timestamp index: %w", err)
	}
	// It seems we have to use FTS4 instead of FTS5? - I could not find an option equivalent to order=DESC for FTS5 and order=DESC makes queries 8-9x faster...
	_, err = db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS EventRaws USING fts4 (raw TEXT, source TEXT, host TEXT, order=DESC);")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws table: %w", err)
	}
	return &sqliteRepository{
		db: db,
	}, nil
}

func (repo *sqliteRepository) AddBatch(events []Event) ([]int64, error) {
	startTime := time.Now()
	ret := make([]int64, len(events))
	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	numberOfDuplicates := map[string]int64{}
	for i, evt := range events {
		res, err := tx.Exec("INSERT INTO Events(host, source, timestamp, offset) VALUES(?, ?, ?, ?);", evt.Host, evt.Source, evt.Timestamp, evt.Offset)
		// Surely this can't be the right way to check for this error...
		if err != nil && err.Error() == expectedConstraintViolationForDuplicates {
			numberOfDuplicates[evt.Source]++
			continue
		}
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error executing add statement: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error getting event id after insert: %w", err)
		}
		_, err = tx.Exec("INSERT INTO EventRaws (rowid, raw, source, host) SELECT LAST_INSERT_ROWID(), ?, ?, ?;", evt.Raw, evt.Source, evt.Host)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error executing add raw statement: %w", err)
		}
		ret[i] = id
	}
	err = tx.Commit()
	if err != nil {
		// TODO: Hmm?
	}
	for k, v := range numberOfDuplicates {
		log.Printf("Skipped adding numEvents=%v from source=%v because they appear to be duplicates (same source, offset and timestamp as an existing event)\n", v, k)
	}
	log.Printf("added numEvents=%v in timeInMs=%v\n", len(events), time.Now().Sub(startTime).Milliseconds())
	return ret, nil
}

func (repo *sqliteRepository) FilterStream(srch *search.Search, searchStartTime, searchEndTime *time.Time) <-chan []EventWithId {
	startTime := time.Now()
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
		var lastTimestamp string
		for {
			stmt := "SELECT e.id, e.host, e.source, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id <= " + strconv.Itoa(maxID)
			if searchStartTime != nil {
				stmt += " AND e.timestamp >= '" + searchStartTime.String() + "'"
			}
			if searchEndTime != nil {
				stmt += " AND e.timestamp <= '" + searchEndTime.String() + "'"
			}
			if lastTimestamp != "" {
				stmt += " AND e.timestamp < '" + lastTimestamp + "'"
			}
			includes := map[string][]string{}
			nots := map[string][]string{}
			hostIncludes := make([]string, 0, len(srch.Hosts))
			for h := range srch.Hosts {
				hostIncludes = append(hostIncludes, h)
			}
			includes["host"] = hostIncludes
			hostNots := make([]string, 0, len(srch.NotHosts))
			for h := range srch.NotHosts {
				hostNots = append(hostNots, h)
			}
			nots["host"] = hostNots
			sourceIncludes := make([]string, 0, len(srch.Sources))
			for s := range srch.Sources {
				sourceIncludes = append(sourceIncludes, s)
			}
			includes["source"] = sourceIncludes
			sourceNots := make([]string, 0, len(srch.NotSources))
			for s := range srch.NotSources {
				sourceNots = append(sourceNots, s)
			}
			nots["source"] = sourceNots
			rawIncludes := make([]string, 0, len(srch.Fragments))
			for f := range srch.Fragments {
				rawIncludes = append(rawIncludes, f)
			}
			includes["raw"] = rawIncludes
			rawNots := make([]string, 0, len(srch.NotFragments))
			for f := range srch.NotFragments {
				rawNots = append(rawNots, f)
			}
			nots["raw"] = rawNots

			matchString := ""
			for k, v := range includes {
				for _, s := range v {
					matchString += k + ":" + s + " "
				}
			}
			for k, v := range nots {
				for _, s := range v {
					matchString += "NOT " + k + ":" + s + " "
				}
			}

			if len(matchString) > 0 {
				stmt += " AND EventRaws MATCH '" + matchString + "'"
			}
			stmt += " ORDER BY e.timestamp DESC LIMIT " + strconv.Itoa(filterStreamPageSize)
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
				err := res.Scan(&evt.Id, &evt.Host, &evt.Source, &evt.Timestamp, &evt.Raw)
				if err != nil {
					log.Printf("error when scanning result in FilterStream: %v\n", err)
				} else {
					evts = append(evts, evt)
				}
				eventsInPage++
				lastTimestamp = evt.Timestamp.String()
			}
			res.Close()
			ret <- evts
			if eventsInPage < filterStreamPageSize {
				endTime := time.Now()
				log.Printf("SQL search completed in timeInMs=%v", endTime.Sub(startTime))
				return
			}
		}
	}()
	return ret
}

func (repo *sqliteRepository) GetByIds(ids []int64, sortMode SortMode) ([]EventWithId, error) {
	ret := make([]EventWithId, len(ids))

	// TODO: I'm PRETTY sure this code is garbage
	stmt := "SELECT e.id, e.host, e.source, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id IN ("
	for i, id := range ids {
		if i == len(ids)-1 {
			stmt += strconv.FormatInt(id, 10)
		} else {
			stmt += strconv.FormatInt(id, 10) + ","
		}
	}
	stmt += ")"

	if sortMode == SortModeTimestampDesc {
		stmt += " ORDER BY e.timestamp DESC;"
	} else {
		stmt += ";"
	}

	res, err := repo.db.Query(stmt)
	if err != nil {
		return nil, fmt.Errorf("error executing GetByIds query: %w", err)
	}
	defer res.Close()

	idx := 0
	for res.Next() {
		err = res.Scan(&ret[idx].Id, &ret[idx].Host, &ret[idx].Source, &ret[idx].Timestamp, &ret[idx].Raw)
		if err != nil {
			return nil, fmt.Errorf("error when scanning row in GetByIds: %w", err)
		}
		idx++
	}

	return ret, nil
}
