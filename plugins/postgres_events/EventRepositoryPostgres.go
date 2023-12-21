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

package postgres_events

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/search"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/dig"
)

const filterStreamPageSize = 1000

type postgresEventRepository struct {
	conn *pgxpool.Pool

	logger *slog.Logger
}

type PostgresEventRepositoryParams struct {
	dig.In

	Conn   *pgxpool.Pool
	Cfg    *config.Config
	Logger *slog.Logger
}

func NewPostgresEventRepository(p PostgresEventRepositoryParams) (events.Repository, error) {
	_, err := p.Conn.Exec(context.TODO(), "CREATE TABLE IF NOT EXISTS Events (id BIGSERIAL NOT NULL PRIMARY KEY, host TEXT NOT NULL, source TEXT NOT NULL, source_id TEXT NOT NULL, timestamp Timestamp NOT NULL, \"offset\" BIGINT NOT NULL, UNIQUE(host, source, timestamp, \"offset\"));")
	if err != nil {
		return nil, fmt.Errorf("error creating events table: %w", err)
	}
	_, err = p.Conn.Exec(context.TODO(), "CREATE INDEX IF NOT EXISTS IX_Events_Timestamp ON Events(timestamp);")
	if err != nil {
		return nil, fmt.Errorf("error creating events timestamp index: %w", err)
	}

	_, err = p.Conn.Exec(context.TODO(), "CREATE TABLE IF NOT EXISTS EventRaws (event_id BIGINT NOT NULL PRIMARY KEY, raw TEXT);")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws table: %w", err)
	}
	_, err = p.Conn.Exec(context.TODO(), "CREATE INDEX IF NOT EXISTS IX_EventRaws_Raw ON EventRaws USING GIN(to_tsvector('english', raw));")
	if err != nil {
		return nil, fmt.Errorf("error creating eventraws index: %w", err)
	}
	return &postgresEventRepository{
		conn:   p.Conn,
		logger: p.Logger,
	}, nil
}

func (repo *postgresEventRepository) AddBatch(events []events.Event) error {
	startTime := time.Now()
	tx, err := repo.conn.BeginTx(context.TODO(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("error starting transaction for adding event: %w", err)
	}
	numberOfDuplicates := map[string]int64{}
	for _, evt := range events {
		res := tx.QueryRow(context.TODO(), "INSERT INTO Events(host, source, source_id, timestamp, \"offset\") VALUES($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING RETURNING id;", evt.Host, evt.Source, evt.SourceId, evt.Timestamp, evt.Offset)
		// Surely this can't be the right way to check for this error...
		/*
			if err != nil && err.Error() == expectedConstraintViolationForDuplicates {
				numberOfDuplicates[evt.Source]++
				continue
			}
		*/
		var id int64
		err := res.Scan(&id)
		if err == pgx.ErrNoRows {
			numberOfDuplicates[evt.Source]++
			continue
		}
		if err != nil {
			tx.Rollback(context.TODO())
			return fmt.Errorf("error executing add statement: %w", err)
		}
		_, err = tx.Exec(context.TODO(), "INSERT INTO EventRaws (event_id, raw) VALUES ($1, $2);", id, evt.Raw)
		if err != nil {
			tx.Rollback(context.TODO())
			return fmt.Errorf("error executing add raw statement: %w", err)
		}
	}
	err = tx.Commit(context.TODO())
	if err != nil {
		// TODO: Hmm?
	}
	for k, v := range numberOfDuplicates {
		repo.logger.Info("Skipped adding events because they appear to be duplicates (same source, offset and timestamp as an existing event)",
			slog.Int64("numEvents", v), slog.String("source", k))
	}
	repo.logger.Info("added events",
		slog.Int("numEvents", len(events)),
		slog.Duration("duration", time.Since(startTime)))
	return nil
}

var delSbBase = "DELETE FROM Events WHERE ID IN ("
var delRawSbBase = "DELETE FROM EventRaws WHERE event_id IN ("
var delSbSuffix = ")"

func (repo *postgresEventRepository) DeleteBatch(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var sb strings.Builder
	var rsb strings.Builder
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

	tx, err := repo.conn.Begin(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to start transaction when deleting numIds=%v: %w", len(ids), err)
	}

	_, err = tx.Exec(context.TODO(), deleteQuery, args...)
	if err != nil {
		tx.Rollback(context.TODO())
		return fmt.Errorf("failed to delete numIds=%v from Events table: %w", len(ids), err)
	}
	_, err = tx.Exec(context.TODO(), deleteRawQuery, args...)
	if err != nil {
		tx.Rollback(context.TODO())
		return fmt.Errorf("failed to delete numIds=%v from EventRaws table: %w", len(ids), err)
	}
	err = tx.Commit(context.TODO())
	if err != nil {
		// I have no idea what you are supposed to do here. rollback or nah?
		return fmt.Errorf("failed to commit delete of numIds=%v. unknown state: %w", len(ids), err)
	}
	return nil
}

func (repo *postgresEventRepository) FilterStream(srch *search.Search, searchStartTime, searchEndTime *time.Time) <-chan []events.EventWithId {
	startTime := time.Now()
	ret := make(chan []events.EventWithId)
	go func() {
		defer close(ret)
		resRow := repo.conn.QueryRow(context.TODO(), "SELECT MAX(id) FROM Events;")
		var maxID int64
		err := resRow.Scan(&maxID)
		if err != nil {
			repo.logger.Error("error when getting max(id) from Events table in FilterStream", slog.Any("error", err))
			return
		}
		var lastTimestamp string
		for {
			stmt := "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.event_id = e.id WHERE e.id <= " + strconv.FormatInt(maxID, 10)
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
			repo.logger.Info("executing SQL statement", slog.String("stmt", stmt))
			res, err := repo.conn.Query(context.TODO(), stmt)
			if err != nil {
				repo.logger.Error("error when getting filtered events in FilterStream", slog.Any("error", err))
				return
			}
			evts := make([]events.EventWithId, 0, filterStreamPageSize)
			eventsInPage := 0

			for res.Next() {
				var evt events.EventWithId
				err := res.Scan(&evt.Id, &evt.Host, &evt.Source, &evt.SourceId, &evt.Timestamp, &evt.Raw)
				if err != nil {
					repo.logger.Warn("error when scanning result in FilterStream", slog.Any("error", err))
				} else {
					evts = append(evts, evt)
				}
				eventsInPage++
				lastTimestamp = evt.Timestamp.String()
			}
			ret <- evts
			if eventsInPage < filterStreamPageSize {
				endTime := time.Now()
				repo.logger.Info("SQL search completed",
					slog.Duration("duration", endTime.Sub(startTime)))
				return
			}
		}
	}()
	return ret
}

func (repo *postgresEventRepository) GetByIds(ids []int64, sortMode events.SortMode) ([]events.EventWithId, error) {
	ret := make([]events.EventWithId, 0, len(ids))

	// TODO: I'm PRETTY sure this code is garbage
	stmt := "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.event_id = e.id WHERE e.id IN ("
	for i, id := range ids {
		if i == len(ids)-1 {
			stmt += strconv.FormatInt(id, 10)
		} else {
			stmt += strconv.FormatInt(id, 10) + ","
		}
	}
	stmt += ")"

	if sortMode == events.SortModeTimestampDesc {
		stmt += " ORDER BY e.timestamp DESC;"
	} else {
		stmt += ";"
	}

	res, err := repo.conn.Query(context.TODO(), stmt)
	if err != nil {
		return nil, fmt.Errorf("error executing GetByIds query: %w", err)
	}
	defer res.Close()

	idx := 0
	for res.Next() {
		ret = append(ret, events.EventWithId{})
		err = res.Scan(&ret[idx].Id, &ret[idx].Host, &ret[idx].Source, &ret[idx].SourceId, &ret[idx].Timestamp, &ret[idx].Raw)
		if err != nil {
			return nil, fmt.Errorf("error when scanning row in GetByIds: %w", err)
		}
		idx++
	}

	if sortMode == events.SortModePreserveArgOrder {
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
const surroundingUpSQL = "SELECT e.id, e.host, e.source, e.source_id, e.timestamp, r.raw FROM Events e INNER JOIN EventRaws r ON r.event_id = e.id WHERE e.source_id=? AND e.offset<=? ORDER BY e.offset DESC LIMIT ?"
const surroundingDownSQL = "SELECT id, host, source, source_id, timestamp, raw FROM (SELECT e.id, e.host, e.source, e.source_id, e.timestamp, e.offset, r.raw FROM Events e INNER JOIN EventRaws r ON r.event_id = e.id WHERE e.source_id=? AND e.offset>? ORDER BY e.offset ASC LIMIT ?) ORDER BY offset DESC"

func (repo *postgresEventRepository) GetSurroundingEvents(id int64, count int) ([]events.EventWithId, error) {
	row := repo.conn.QueryRow(context.TODO(), surroundingBaseSQL, id)
	var sourceId string
	var baseOffset int
	err := row.Scan(&sourceId, &baseOffset)
	if err != nil {
		return nil, fmt.Errorf("got error when scanning source_id and offset for eventId=%v: %w", id, err)
	}

	upRows, err := queryAndScan(repo.conn, surroundingUpSQL, sourceId, baseOffset, count/2)
	if err != nil {
		return nil, fmt.Errorf("got error when querying for surrounding rows for eventId=%v: %w", id, err)
	}
	downRows, err := queryAndScan(repo.conn, surroundingDownSQL, sourceId, baseOffset, count/2)
	if err != nil {
		return nil, fmt.Errorf("got error when querying for surrounding rows for eventId=%v: %w", id, err)
	}

	return append(downRows, upRows...), nil
}

func queryAndScan(conn *pgxpool.Pool, query string, sourceId string, baseOffset int, count int) ([]events.EventWithId, error) {
	ret := make([]events.EventWithId, 0, count)
	rows, err := conn.Query(context.TODO(), query, sourceId, baseOffset, count)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var evt events.EventWithId
		err := rows.Scan(&evt.Id, &evt.Host, &evt.Source, &evt.SourceId, &evt.Timestamp, &evt.Raw)
		if err != nil {
			return nil, err
		}
		ret = append(ret, evt)
	}
	return ret, nil
}
