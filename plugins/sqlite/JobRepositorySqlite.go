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

package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/jobs"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type sqliteJobRepository struct {
	db *sql.DB
}

func NewSqliteJobRepository(db *sql.DB) (jobs.Repository, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Jobs (id INTEGER NOT NULL PRIMARY KEY, state INTEGER NOT NULL, query TEXT NOT NULL, start_time DATETIME, end_time DATETIME, sort_mode INTEGER NOT NULL, output_type INTEGER NOT NULL, column_order_json TEXT NOT NULL);")
	if err != nil {
		return nil, fmt.Errorf("error when creating Jobs table: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS JobResults (job_id INTEGER NOT NULL, event_id INTEGER NOT NULL, timestamp DATETIME NOT NULL, FOREIGN KEY(job_id) REFERENCES Jobs(id), FOREIGN KEY(event_id) REFERENCES Events(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobResults table: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS JobTableResults (job_id INTEGER NOT NULL, row_number INTEGER NOT NULL, row_json TEXT NOT NULL, FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobTableResults table: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS JobFieldValues (job_id INTEGER NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL, occurrences INTEGER NOT NULL, UNIQUE(job_id, key, value), FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobFieldValues table: %w", err)
	}
	return &sqliteJobRepository{
		db: db,
	}, nil
}

func (repo *sqliteJobRepository) AddResults(id int64, events []events.EventIdAndTimestamp) error {
	if len(events) == 0 {
		return nil
	}
	idString := strconv.FormatInt(id, 10)
	stmt := "INSERT INTO JobResults (job_id, event_id, timestamp) VALUES "
	for i, evt := range events {
		stmt += "(" + idString + ", " + strconv.FormatInt(evt.Id, 10) + ", '" + evt.Timestamp.String() + "')"
		if i != len(events)-1 {
			stmt += ", "
		}
	}
	stmt += ";"
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return fmt.Errorf("error adding results to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *sqliteJobRepository) AddTableResults(id int64, tableRows []jobs.TableRow) error {
	if len(tableRows) == 0 {
		return nil
	}
	idString := strconv.FormatInt(id, 10)
	stmt := "INSERT INTO JobTableResults (job_id, row_number, row_json) VALUES "
	for i, r := range tableRows {
		b, err := json.Marshal(r.Values)
		if err != nil {
			return fmt.Errorf("error adding table results to jobId=%v: failed to marshal row=%v: %w", id, r, err)
		}
		stmt += "(" + idString + ", " + strconv.FormatInt(int64(r.RowNumber), 10) + ", '" + string(b) + "')"
		if i != len(tableRows)-1 {
			stmt += ", "
		}
	}
	stmt += ";"
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return fmt.Errorf("error adding results to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *sqliteJobRepository) AddFieldStats(id int64, fields []jobs.FieldStats) error {
	idString := strconv.FormatInt(id, 10)
	stmt := "INSERT INTO JobFieldValues (job_id, key, value, occurrences) VALUES "
	for i, f := range fields {
		stmt += "(" + idString + ", '" + f.Key + "', '" + f.Value + "', " + strconv.Itoa(f.Occurrences) + ")"
		if i != len(fields)-1 {
			stmt += ", "
		}
	}
	stmt += " ON CONFLICT (job_id, key, value) DO UPDATE SET occurrences = occurrences + excluded.occurrences;"
	_, err := repo.db.Exec(stmt)
	if err != nil {
		return fmt.Errorf("error when adding stats to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *sqliteJobRepository) Get(id int64) (*jobs.Job, error) {
	res, err := repo.db.Query("SELECT id, state, query, start_time, end_time, sort_mode, output_type, column_order_json FROM Jobs WHERE id=?;", id)
	if err != nil {
		return nil, fmt.Errorf("error getting job with jobId=%v: %w", id, err)
	}
	defer res.Close()
	if !res.Next() {
		return nil, fmt.Errorf("jobId=%v not found", id)
	}
	var job jobs.Job
	var columnOrderJson string
	err = res.Scan(&job.Id, &job.State, &job.Query, &job.StartTime, &job.EndTime, &job.SortMode, &job.OutputType, &columnOrderJson)
	if err != nil {
		return nil, fmt.Errorf("error reading jobId=%v from database: %w", id, err)
	}
	err = json.Unmarshal([]byte(columnOrderJson), &job.ColumnOrder)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling column order JSON for jobId=%v, columnOrderJson=%v: %w", id, columnOrderJson, err)
	}
	return &job, nil
}

func (repo *sqliteJobRepository) GetResults(jobId int64, skip int, take int) ([]int64, error) {
	res, err := repo.db.Query("SELECT event_id FROM JobResults WHERE job_id=? ORDER BY timestamp DESC LIMIT ? OFFSET ?;", jobId, take, skip)
	if err != nil {
		return nil, fmt.Errorf("error when getting results for jobId=%v, skip=%v, take=%v: %w", jobId, skip, take, err)
	}
	defer res.Close()
	ids := make([]int64, 0, take)
	for res.Next() {
		var id int64
		err = res.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("error reading event id from database when getting results for jobId=%v, skip=%v, take=%v: %w", jobId, skip, take, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (repo *sqliteJobRepository) GetTableResults(id int64, skip int, take int) ([]jobs.TableRow, error) {
	res, err := repo.db.Query("SELECT row_number, row_json FROM JobTableResults WHERE job_id=? ORDER BY row_number LIMIT ? OFFSET ?", id, take, skip)
	if err != nil {
		return nil, fmt.Errorf("error when getting table results for jobId=%v, skip=%v, take=%v: %w", id, skip, take, err)
	}
	defer res.Close()
	ret := make([]jobs.TableRow, 0, take+1)
	for res.Next() {
		var rowNumber int
		var rowJson string
		err = res.Scan(&rowNumber, &rowJson)
		if err != nil {
			return nil, fmt.Errorf("error reading table row from database when getting table results for jobId=%v, skip=%v, take=%v: %w", id, skip, take, err)
		}
		var values map[string]string
		err = json.Unmarshal([]byte(rowJson), &values)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling table row when getting table results for jobId=%v, skip=%v, take=%v: %w", id, skip, take, err)
		}
		ret = append(ret, jobs.TableRow{
			RowNumber: rowNumber,
			Values:    values,
		})
	}
	return ret, nil
}

func (repo *sqliteJobRepository) GetFieldOccurences(id int64) (map[string]int, error) {
	res, err := repo.db.Query("SELECT key, COUNT(1) FROM JobFieldValues WHERE job_id=? GROUP BY key;", id)
	if err != nil {
		return nil, fmt.Errorf("error when getting field occurrences for jobId=%v: %w", id, err)
	}
	defer res.Close()
	m := map[string]int{}
	for res.Next() {
		var key string
		var count int
		err = res.Scan(&key, &count)
		if err != nil {
			return nil, fmt.Errorf("error reading field occurences for jobId=%v: %w", id, err)
		}
		m[key] = count
	}
	return m, nil
}

func (repo *sqliteJobRepository) GetFieldValues(id int64, fieldName string) (map[string]int, error) {
	res, err := repo.db.Query("SELECT value, occurrences FROM JobFieldValues WHERE job_id=? AND key=?;", id, fieldName)
	if err != nil {
		return nil, fmt.Errorf("error when getting field values for jobId=%v and fieldName=%v: %w", id, fieldName, err)
	}
	defer res.Close()
	m := map[string]int{}
	for res.Next() {
		var value string
		var count int
		err = res.Scan(&value, &count)
		if err != nil {
			return nil, fmt.Errorf("error reading field values for jobId=%v and fieldName=%v: %w", id, fieldName, err)
		}
		m[value] = count
	}
	return m, nil
}

func (repo *sqliteJobRepository) GetNumMatchedEvents(id int64) (int64, error) {
	job, err := repo.Get(id)
	if err != nil {
		return 0, fmt.Errorf("error when getting number of matched events for jobId=%v: %w", id, err)
	}
	stmt := "SELECT COUNT(1) FROM JobResults WHERE job_id=?"
	if job.OutputType == pipeline.PipelinePipeTypeTable {
		stmt = "SELECT COUNT(1) FROM JobTableResults WHERE job_id=?"
	}
	res, err := repo.db.Query(stmt, id)
	if err != nil {
		return 0, fmt.Errorf("error when getting number of matched events for jobId=%v: %w", id, err)
	}
	defer res.Close()
	if !res.Next() {
		return 0, fmt.Errorf("error when getting number of matched events for jobId=%v, expected at least one row in result set but have 0", id)
	}
	var count int64
	err = res.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error when reading number of matched events for jobId=%v: %w", id, err)
	}
	return count, nil
}

func (repo *sqliteJobRepository) Insert(query string, startTime, endTime *time.Time, sortMode events.SortMode, outputType pipeline.PipelinePipeType, columnOrder []string) (*int64, error) {
	columnOrderJson, err := json.Marshal(columnOrder)
	if err != nil {
		return nil, fmt.Errorf("error when inserting new job: error marshaling columnOrder: %w", err)
	}
	res, err := repo.db.Exec("INSERT INTO Jobs (state, query, start_time, end_time, sort_mode, output_type, column_order_json) VALUES(?, ?, ?, ?, ?, ?, ?);",
		jobs.JobStateRunning, query, startTime, endTime, sortMode, outputType, columnOrderJson)
	if err != nil {
		return nil, fmt.Errorf("error when inserting new job: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		// This is a pretty bad situation to be in, the job will currently just be stuck in running state forever in the table.
		return nil, fmt.Errorf("error when getting ID of newly inserted job: %w", err)
	}
	return &id, nil
}

func (repo *sqliteJobRepository) UpdateState(id int64, state jobs.JobState) error {
	_, err := repo.db.Exec("UPDATE Jobs SET state=? WHERE id=?;", state, id)
	if err != nil {
		return fmt.Errorf("error when updating jobId=%v to state=%v: %w", id, state, err)
	}
	return nil
}
