package jobs

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/internal/events"
)

type sqliteRepository struct {
	db *sql.DB
}

func SqliteRepository(db *sql.DB) (*sqliteRepository, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Jobs (id INTEGER NOT NULL PRIMARY KEY, state INTEGER NOT NULL, query TEXT NOT NULL, start_time DATETIME, end_time DATETIME);")
	if err != nil {
		return nil, fmt.Errorf("error when creating Jobs table: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS JobResults (job_id INTEGER NOT NULL, event_id INTEGER NOT NULL, timestamp DATETIME NOT NULL, FOREIGN KEY(job_id) REFERENCES Jobs(id), FOREIGN KEY(event_id) REFERENCES Events(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobResults table: %w", err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS JobFieldValues (job_id INTEGER NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL, occurrences INTEGER NOT NULL, UNIQUE(job_id, key, value), FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobFieldValues table: %w", err)
	}
	return &sqliteRepository{
		db: db,
	}, nil
}

func (repo *sqliteRepository) AddResults(id int64, events []events.EventIdAndTimestamp) error {
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

func (repo *sqliteRepository) AddFieldStats(id int64, fields []FieldStats) error {
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

func (repo *sqliteRepository) Get(id int64) (*Job, error) {
	res, err := repo.db.Query("SELECT id, state, query, start_time, end_time FROM Jobs WHERE id=?;", id)
	if err != nil {
		return nil, fmt.Errorf("error getting job with jobId=%v: %w", id, err)
	}
	defer res.Close()
	if !res.Next() {
		return nil, fmt.Errorf("jobId=%v not found", id)
	}
	var job Job
	err = res.Scan(&job.Id, &job.State, &job.Query, &job.StartTime, &job.EndTime)
	if err != nil {
		return nil, fmt.Errorf("error reading jobId=%v from database: %w", id, err)
	}
	return &job, nil
}

func (repo *sqliteRepository) GetResults(jobId int64, skip int, take int) ([]int64, error) {
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

func (repo *sqliteRepository) GetFieldOccurences(id int64) (map[string]int, error) {
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

func (repo *sqliteRepository) GetFieldValues(id int64, fieldName string) (map[string]int, error) {
	res, err := repo.db.Query("SELECT value, occurrences FROM JobFieldValues WHERE job_id=? AND key=?;", id, fieldName)
	if err != nil {
		return nil, fmt.Errorf("error when getting field values for jobId=%v and fieldName=%v: %w", id, fieldName, err)
	}
	defer res.Close()
	var m map[string]int
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

func (repo *sqliteRepository) GetNumMatchedEvents(id int64) (int64, error) {
	res, err := repo.db.Query("SELECT COUNT(1) FROM JobResults WHERE job_id=?;", id)
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

func (repo *sqliteRepository) Insert(query string, startTime, endTime *time.Time) (*int64, error) {
	res, err := repo.db.Exec("INSERT INTO Jobs (state, query, start_time, end_time) VALUES(?, ?, ?, ?);",
		JobStateRunning, query, startTime, endTime)
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

func (repo *sqliteRepository) UpdateState(id int64, state JobState) error {
	_, err := repo.db.Exec("UPDATE Jobs SET state=? WHERE id=?;", id, state)
	if err != nil {
		return fmt.Errorf("error when updating jobId=%v to state=%v: %w", id, state, err)
	}
	return nil
}
