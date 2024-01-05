package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/jobs"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
)

type PostgresJobRepository struct {
	pool *pgxpool.Pool
}

type PostgresJobRepositoryParams struct {
	dig.In

	Ctx  context.Context
	Pool *pgxpool.Pool
}

func NewPostgresJobRepository(p PostgresJobRepositoryParams) (jobs.Repository, error) {
	_, err := p.Pool.Exec(p.Ctx, "CREATE TABLE IF NOT EXISTS Jobs (id SERIAL NOT NULL PRIMARY KEY, state INTEGER NOT NULL, query TEXT NOT NULL, start_time TIMESTAMP, end_time TIMESTAMP, sort_mode INTEGER NOT NULL, output_type INTEGER NOT NULL, column_order_json JSONB NOT NULL);")
	if err != nil {
		return nil, fmt.Errorf("error when creating Jobs table: %w", err)
	}
	_, err = p.Pool.Exec(p.Ctx, "CREATE TABLE IF NOT EXISTS JobResults (job_id INTEGER NOT NULL, event_id INTEGER NOT NULL, timestamp TIMESTAMP NOT NULL, FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobResults table: %w", err)
	}
	_, err = p.Pool.Exec(p.Ctx, "CREATE TABLE IF NOT EXISTS JobTableResults (job_id INTEGER NOT NULL, row_number INTEGER NOT NULL, row_json TEXT NOT NULL, FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobTableResults table: %w", err)
	}
	_, err = p.Pool.Exec(p.Ctx, "CREATE TABLE IF NOT EXISTS JobFieldValues (job_id INTEGER NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL, occurrences INTEGER NOT NULL, UNIQUE(job_id, key, value), FOREIGN KEY(job_id) REFERENCES Jobs(id));")
	if err != nil {
		return nil, fmt.Errorf("error when creating JobFieldValues table: %w", err)
	}
	return &PostgresJobRepository{
		pool: p.Pool,
	}, nil
}

func (repo *PostgresJobRepository) AddResults(id int64, events []events.EventIdAndTimestamp) error {
	if len(events) == 0 {
		return nil
	}
	idString := strconv.FormatInt(id, 10)
	stmt := "INSERT INTO JobResults (job_id, event_id, timestamp) VALUES "
	for i, evt := range events {
		stmt += "(" + idString + ", " + strconv.FormatInt(evt.Id, 10) + ", '" + evt.Timestamp.Format(time.RFC3339Nano) + "')"
		if i != len(events)-1 {
			stmt += ", "
		}
	}
	stmt += ";"
	_, err := repo.pool.Exec(context.TODO(), stmt)
	if err != nil {
		return fmt.Errorf("error adding results to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *PostgresJobRepository) AddTableResults(id int64, tableRows []jobs.TableRow) error {
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
	_, err := repo.pool.Exec(context.TODO(), stmt)
	if err != nil {
		return fmt.Errorf("error adding results to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *PostgresJobRepository) AddFieldStats(id int64, fields []jobs.FieldStats) error {
	idString := strconv.FormatInt(id, 10)
	batch := &pgx.Batch{}
	for _, f := range fields {
		batch.Queue("INSERT INTO JobFieldValues AS jfv (job_id, key, value, occurrences) VALUES ($1, $2, $3, $4) ON CONFLICT (job_id, key, value) DO UPDATE SET occurrences = jfv.occurrences + excluded.occurrences;",
			idString, f.Key, f.Value, f.Occurrences)
	}
	res := repo.pool.SendBatch(context.TODO(), batch)
	defer res.Close()
	_, err := res.Exec()
	if err != nil {
		return fmt.Errorf("error when adding stats to jobId=%v: %w", id, err)
	}
	return nil
}

func (repo *PostgresJobRepository) Get(id int64) (*jobs.Job, error) {
	res, err := repo.pool.Query(context.TODO(), "SELECT id, state, query, start_time, end_time, sort_mode, output_type, column_order_json FROM Jobs WHERE id=$1;", id)
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

func (repo *PostgresJobRepository) GetResults(jobId int64, skip int, take int) ([]int64, error) {
	res, err := repo.pool.Query(context.TODO(), "SELECT event_id FROM JobResults WHERE job_id=$1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3;", jobId, take, skip)
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

func (repo *PostgresJobRepository) GetTableResults(id int64, skip int, take int) ([]jobs.TableRow, error) {
	res, err := repo.pool.Query(context.TODO(), "SELECT row_number, row_json FROM JobTableResults WHERE job_id=$1 ORDER BY row_number LIMIT $2 OFFSET $3", id, take, skip)
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

func (repo *PostgresJobRepository) GetFieldOccurences(id int64) (map[string]int, error) {
	res, err := repo.pool.Query(context.TODO(), "SELECT key, COUNT(1) FROM JobFieldValues WHERE job_id=$1 GROUP BY key;", id)
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

func (repo *PostgresJobRepository) GetFieldValues(id int64, fieldName string) (map[string]int, error) {
	res, err := repo.pool.Query(context.TODO(), "SELECT value, occurrences FROM JobFieldValues WHERE job_id=$1 AND key=$2;", id, fieldName)
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

func (repo *PostgresJobRepository) GetNumMatchedEvents(id int64) (int64, error) {
	job, err := repo.Get(id)
	if err != nil {
		return 0, fmt.Errorf("error when getting number of matched events for jobId=%v: %w", id, err)
	}
	stmt := "SELECT COUNT(1) FROM JobResults WHERE job_id=$1"
	if job.OutputType == pipeline.PipeTypeTable {
		stmt = "SELECT COUNT(1) FROM JobTableResults WHERE job_id=$1"
	}
	res, err := repo.pool.Query(context.TODO(), stmt, id)
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

func (repo *PostgresJobRepository) Insert(query string, startTime, endTime *time.Time, sortMode events.SortMode, outputType pipeline.PipeType, columnOrder []string) (*int64, error) {
	columnOrderJson, err := json.Marshal(columnOrder)
	if err != nil {
		return nil, fmt.Errorf("error when inserting new job: error marshaling columnOrder: %w", err)
	}
	res := repo.pool.QueryRow(context.TODO(), "INSERT INTO Jobs (state, query, start_time, end_time, sort_mode, output_type, column_order_json) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id;",
		jobs.StateRunning, query, startTime, endTime, sortMode, outputType, columnOrderJson)
	if err != nil {
		return nil, fmt.Errorf("error when inserting new job: %w", err)
	}
	var id int64
	err = res.Scan(&id)
	if err != nil {
		// This is a pretty bad situation to be in, the job will currently just be stuck in running state forever in the table.
		return nil, fmt.Errorf("error when getting ID of newly inserted job: %w", err)
	}
	return &id, nil
}

func (repo *PostgresJobRepository) UpdateState(id int64, state jobs.State) error {
	_, err := repo.pool.Exec(context.TODO(), "UPDATE Jobs SET state=$1 WHERE id=$2;", state, id)
	if err != nil {
		return fmt.Errorf("error when updating jobId=%v to state=%v: %w", id, state, err)
	}
	return nil
}
