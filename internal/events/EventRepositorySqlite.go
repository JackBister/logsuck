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

package events

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/search"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

// It might be more correct to use source_id + offset for deduplication, but this works poorly in single mode / while developing since
// a new source ID is generated for all existing files on each restart.
const expectedConstraintViolationForDuplicates = "UNIQUE constraint failed: Events.host, Events.source, Events.timestamp, Events.offset"
const expectedErrorWhenDatabaseIsEmpty = "sql: Scan error on column index 0, name \"MAX(id)\": converting NULL to int is unsupported"
const filterStreamPageSize = 1000

type sqliteRepository struct {
	db *sql.DB

	cfg *config.SqliteConfig

	logger *zap.Logger
}

type SqliteEventRepositoryParams struct {
	dig.In

	Db     *sql.DB
	Cfg    *config.Config
	Logger *zap.Logger
}

func SqliteRepository(p SqliteEventRepositoryParams) (Repository, error) {
	_, err := p.Db.Exec("CREATE TABLE IF NOT EXISTS Events (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, host TEXT NOT NULL, source TEXT NOT NULL, source_id TEXT NOT NULL, timestamp DATETIME NOT NULL, offset BIGINT NOT NULL, UNIQUE(host, source, timestamp, offset));")
	if err != nil {
		return nil, fmt.Errorf("error creating events table: %w", err)
	}
	_, err = p.Db.Exec("CREATE INDEX IF NOT EXISTS IX_Events_Timestamp ON Events(timestamp);")
	if err != nil {
		return nil, fmt.Errorf("error creating events timestamp index: %w", err)
	}
	// It seems we have to use FTS4 instead of FTS5? - I could not find an option equivalent to order=DESC for FTS5 and order=DESC makes queries 8-9x faster...
	_, err = p.Db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS EventRaws USING fts4 (raw TEXT, source TEXT, host TEXT, order=DESC);")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws table: %w", err)
	}
	return &sqliteRepository{
		db:     p.Db,
		cfg:    p.Cfg.SQLite,
		logger: p.Logger,
	}, nil
}

func (repo *sqliteRepository) AddBatch(events []Event) error {
	if repo.cfg.TrueBatch {
		return repo.addBatchTrueBatch(events)
	} else {
		return repo.addBatchOneByOne(events)
	}
}

const esbBase = "INSERT OR IGNORE INTO Events (host, source, source_id, timestamp, offset) VALUES "
const esbBaseLen = len(esbBase)
const esbPerEvt = "(?, ?, ?, ?, ?)"
const esbPerEvtLen = len(esbPerEvt)
const rsbBase = "INSERT INTO EventRaws (raw, source, host) VALUES "
const rsbBaseLen = len(rsbBase)
const rsbPerEvt = "(?, ?, ?)"
const rsbPerEvtLen = len(rsbPerEvt)

func (repo *sqliteRepository) addBatchTrueBatch(events []Event) error {
	startTime := time.Now()
	var eventSb strings.Builder
	var rawSb strings.Builder
	eventSb.Grow(esbBaseLen + esbPerEvtLen*len(events) + len(events))
	rawSb.Grow(rsbBaseLen + rsbPerEvtLen*len(events) + len(events))
	eventSb.WriteString(esbBase)
	rawSb.WriteString(rsbBase)

	esbArgs := make([]interface{}, 0, 4*len(events))
	rsbArgs := make([]interface{}, 0, 3*len(events))
	for i, evt := range events {
		eventSb.WriteString(esbPerEvt)
		rawSb.WriteString(rsbPerEvt)
		if i != len(events)-1 {
			eventSb.WriteRune(',')
			rawSb.WriteRune(',')
		}
		esbArgs = append(esbArgs, evt.Host, evt.Source, evt.SourceId, evt.Timestamp, evt.Offset)
		rsbArgs = append(rsbArgs, evt.Raw, evt.Source, evt.Host)
	}

	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("error starting transaction for adding event batch: %w", err)
	}
	rows, err := tx.Query("SELECT MAX(rowid) FROM EventRaws;")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error adding event batch: failed to get MAX(rowid): %w", err)
	}
	prevMaxID := 0
	if rows.Next() {
		rows.Scan(&prevMaxID)
	}
	eventQ := eventSb.String()
	_, err = tx.Exec(eventQ, esbArgs...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error adding event batch to Events table: %w", err)
	}
	rawQ := rawSb.String()
	res, err := tx.Exec(rawQ, rsbArgs...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error adding event batch to EventRaws table: %w", err)
	}
	newMaxID, err := res.LastInsertId()
	if err != nil {
		repo.logger.Error("got error when getting new max ID to clean up EventRaws", zap.Error(err))
	} else {
		res, err = tx.Exec("DELETE FROM EventRaws AS er WHERE NOT EXISTS (SELECT 1 FROM Events e WHERE e.ID = er.rowid) AND er.rowid > ? AND er.rowid <= ? AND er.rowid != (SELECT MAX(ID) FROM Events)", prevMaxID, newMaxID)
		if err != nil {
			repo.logger.Error("got error when cleaning up EventRaws", zap.Error(err))
		} else if deleted, err := res.RowsAffected(); err == nil && deleted > 0 {
			repo.logger.Info("Skipped adding events as they appear to be duplicates (same source, offset and timestamp as an existing event)",
				zap.Int64("numEvents", deleted))
		}
	}
	err = tx.Commit()
	if err != nil {
		// TODO: Hmm?
	}
	repo.logger.Info("added events",
		zap.Int("numEvents", len(events)),
		zap.Stringer("duration", time.Now().Sub(startTime)))
	return nil
}

func (repo *sqliteRepository) addBatchOneByOne(events []Event) error {
	startTime := time.Now()
	ret := make([]int64, len(events))
	tx, err := repo.db.BeginTx(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	numberOfDuplicates := map[string]int64{}
	for i, evt := range events {
		res, err := tx.Exec("INSERT INTO Events(host, source, source_id, timestamp, offset) VALUES(?, ?, ?, ?, ?);", evt.Host, evt.Source, evt.SourceId, evt.Timestamp, evt.Offset)
		// Surely this can't be the right way to check for this error...
		if err != nil && err.Error() == expectedConstraintViolationForDuplicates {
			numberOfDuplicates[evt.Source]++
			continue
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing add statement: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error getting event id after insert: %w", err)
		}
		_, err = tx.Exec("INSERT INTO EventRaws (rowid, raw, source, host) SELECT LAST_INSERT_ROWID(), ?, ?, ?;", evt.Raw, evt.Source, evt.Host)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error executing add raw statement: %w", err)
		}
		ret[i] = id
	}
	err = tx.Commit()
	if err != nil {
		// TODO: Hmm?
	}
	for k, v := range numberOfDuplicates {
		repo.logger.Info("Skipped adding events because they appear to be duplicates (same source, offset and timestamp as an existing event)",
			zap.Int64("numEvents", v), zap.String("source", k))
	}
	repo.logger.Info("added events",
		zap.Int("numEvents", len(events)),
		zap.Stringer("duration", time.Now().Sub(startTime)))
	return nil
}

var delSbBase = "DELETE FROM Events WHERE ID IN ("
var delSbBaseLen = len(delSbBase)
var delRawSbBase = "DELETE FROM EventRaws WHERE rowid IN ("
var delRawSbBaseLen = len(delRawSbBase)
var delSbPerEvtLen = 3 // "?, " except for the last one which is just ?. So the buffer ends up being two bytes too large.
var delSbSuffix = ")"
var delSbSuffixLen = len(delSbSuffix)

func (repo *sqliteRepository) DeleteBatch(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var sb strings.Builder
	var rsb strings.Builder
	sb.Grow(delSbBaseLen + len(ids)*delSbPerEvtLen + delSbSuffixLen)
	rsb.Grow(delRawSbBaseLen + len(ids)*delSbPerEvtLen + delSbSuffixLen)
	sb.WriteString(delSbBase)
	rsb.WriteString(delRawSbBase)
	for i := range ids {
		if i != len(ids)-1 {
			sb.WriteString("?, ")
			rsb.WriteString("?, ")
		} else {
			sb.WriteString("?")
			rsb.WriteString("?")
		}
	}
	sb.WriteString(delSbSuffix)
	rsb.WriteString(delSbSuffix)
	deleteQuery := sb.String()
	deleteRawQuery := rsb.String()

	// yuck
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction when deleting numIds=%v: %w", len(ids), err)
	}

	_, err = tx.Exec(deleteQuery, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete numIds=%v from Events table: %w", len(ids), err)
	}
	_, err = tx.Exec(deleteRawQuery, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete numIds=%v from EventRaws table: %w", len(ids), err)
	}
	err = tx.Commit()
	if err != nil {
		// I have no idea what you are supposed to do here. rollback or nah?
		return fmt.Errorf("failed to commit delete of numIds=%v. unknown state: %w", len(ids), err)
	}
	return nil
}

func (repo *sqliteRepository) FilterStream(srch *search.Search, searchStartTime, searchEndTime *time.Time) <-chan []EventWithId {
	startTime := time.Now()
	ret := make(chan []EventWithId)
	go func() {
		defer close(ret)
		res, err := repo.db.Query("SELECT MAX(id) FROM Events;")
		if err != nil {
			repo.logger.Error("error when getting max(id) from Events table in FilterStream", zap.Error(err))
			return
		}
		if !res.Next() {
			res.Close()
			repo.logger.Error("weird state in FilterStream, expected one result when getting max(id) from Events but got 0")
			return
		}
		var maxID int
		err = res.Scan(&maxID)
		res.Close()
		if err != nil {
			if err.Error() == expectedErrorWhenDatabaseIsEmpty {
				return
			}
			repo.logger.Error("error when scanning max(id) in FilterStream", zap.Error(err))
			return
		}
		var lastTimestamp string
		for {
			stmt := "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id <= " + strconv.Itoa(maxID)
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
			repo.logger.Info("executing SQL statement", zap.String("stmt", stmt))
			res, err = repo.db.Query(stmt)
			if err != nil {
				repo.logger.Error("error when getting filtered events in FilterStream", zap.Error(err))
				return
			}
			evts := make([]EventWithId, 0, filterStreamPageSize)
			eventsInPage := 0
			for res.Next() {
				var evt EventWithId
				err := res.Scan(&evt.Id, &evt.Host, &evt.Source, &evt.SourceId, &evt.Timestamp, &evt.Raw)
				if err != nil {
					repo.logger.Warn("error when scanning result in FilterStream", zap.Error(err))
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
				repo.logger.Info("SQL search completed",
					zap.Stringer("duration", endTime.Sub(startTime)))
				return
			}
		}
	}()
	return ret
}

func (repo *sqliteRepository) GetByIds(ids []int64, sortMode SortMode) ([]EventWithId, error) {
	ret := make([]EventWithId, 0, len(ids))

	// TODO: I'm PRETTY sure this code is garbage
	stmt := "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.id IN ("
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
		ret = append(ret, EventWithId{})
		err = res.Scan(&ret[idx].Id, &ret[idx].Host, &ret[idx].Source, &ret[idx].SourceId, &ret[idx].Timestamp, &ret[idx].Raw)
		if err != nil {
			return nil, fmt.Errorf("error when scanning row in GetByIds: %w", err)
		}
		idx++
	}

	if sortMode == SortModePreserveArgOrder {
		m := make(map[int64]int, len(ids))
		for i, id := range ids {
			m[id] = i
		}
		sort.Slice(ret, func(i, j int) bool {
			return m[ret[i].Id] < m[ret[j].Id]
		})
	}

	return ret, nil
}

const surroundingBaseSQL = "SELECT source_id, offset FROM Events WHERE id=?"
const surroundingUpSQL = "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.source_id=? AND e.offset<=? ORDER BY e.offset DESC LIMIT ?"
const surroundingDownSQL = "SELECT id, host, source, source_id, timestamp, raw FROM (SELECT e.id, e.host, e.source, e.source_id, e.timestamp, e.offset, r.raw FROM Events e INNER JOIN EventRaws r ON r.rowid = e.id WHERE e.source_id=? AND e.offset>? ORDER BY e.offset ASC LIMIT ?) ORDER BY offset DESC"

func (repo *sqliteRepository) GetSurroundingEvents(id int64, count int) ([]EventWithId, error) {
	row := repo.db.QueryRow(surroundingBaseSQL, id)
	if row == nil {
		return []EventWithId{}, nil
	}
	if row.Err() != nil {
		return nil, fmt.Errorf("got error when getting source_id and offset for eventId=%v: %w", id, row.Err())
	}

	var sourceId string
	var baseOffset int
	err := row.Scan(&sourceId, &baseOffset)
	if err != nil {
		return nil, fmt.Errorf("got error when scanning source_id and offset for eventId=%v: %w", id, err)
	}

	upRows, err := queryAndScan(repo.db, surroundingUpSQL, sourceId, baseOffset, count/2)
	if err != nil {
		return nil, fmt.Errorf("got error when querying for surrounding rows for eventId=%v: %w", id, err)
	}
	downRows, err := queryAndScan(repo.db, surroundingDownSQL, sourceId, baseOffset, count/2)

	return append(downRows, upRows...), nil
}

func queryAndScan(db *sql.DB, query string, sourceId string, baseOffset int, count int) ([]EventWithId, error) {
	ret := make([]EventWithId, 0, count)
	rows, err := db.Query(query, sourceId, baseOffset, count)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var evt EventWithId
		err := rows.Scan(&evt.Id, &evt.Host, &evt.Source, &evt.SourceId, &evt.Timestamp, &evt.Raw)
		if err != nil {
			return nil, err
		}
		ret = append(ret, evt)
	}
	return ret, nil
}
