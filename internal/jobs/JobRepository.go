package jobs

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/emirpasic/gods/sets/treeset"
	"github.com/jackbister/logsuck/internal/events"
)

type Repository interface {
	AddResults(id int64, events []events.EventIdAndTimestamp) error
	AddFieldStats(id int64, fields []FieldStats) error
	Get(id int64) (*Job, error)
	GetResults(id int64, skip int, take int) (eventIds []int64, err error)
	GetFieldOccurences(id int64) (map[string]int, error)
	GetFieldValues(id int64, fieldName string) (map[string]int, error)
	GetNumMatchedEvents(id int64) (int64, error)
	Insert(query string, startTime, endTime *time.Time) (id *int64, err error)
	UpdateState(id int64, state JobState) error
}

type FieldStats struct {
	Key         string
	Value       string
	Occurrences int
}

type inMemoryRepository struct {
	jobs        map[int64]*Job
	results     map[int64]*treeset.Set // tree of EventIdAndTimestamp ordered in descending timestamp order
	stats       map[int64]*JobStats
	statMutexes map[int64]*sync.RWMutex
}

func InMemoryRepository() Repository {
	return &inMemoryRepository{
		jobs:        map[int64]*Job{},
		results:     map[int64]*treeset.Set{},
		stats:       map[int64]*JobStats{},
		statMutexes: map[int64]*sync.RWMutex{},
	}
}

func resultComparator(a, b interface{}) int {
	aEvt := a.(events.EventIdAndTimestamp)
	bEvt := b.(events.EventIdAndTimestamp)
	timeDiff := bEvt.Timestamp.Sub(aEvt.Timestamp).Milliseconds()
	if timeDiff == 0 {
		return int(bEvt.Id - aEvt.Id)
	} else {
		return int(timeDiff)
	}
}

func (repo *inMemoryRepository) AddResult(id int64, event events.EventIdAndTimestamp) error {
	stats, ok := repo.stats[id]
	if !ok {
		return errors.New("job with Id=" + string(id) + " not found")
	}
	if _, ok := repo.results[id]; !ok {
		repo.results[id] = treeset.NewWith(resultComparator)
	}
	repo.results[id].Add(event)
	stats.NumMatchedEvents++
	return nil
}

func (repo *inMemoryRepository) AddResults(id int64, events []events.EventIdAndTimestamp) error {
	for _, evt := range events {
		err := repo.AddResult(id, evt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (repo *inMemoryRepository) AddFieldStats(id int64, fields []FieldStats) error {
	stats, ok := repo.stats[id]
	if !ok {
		return errors.New("job with Id=" + string(rune(id)) + " not found")
	}
	repo.statMutexes[id].Lock()
	defer repo.statMutexes[id].Unlock()
	for _, f := range fields {
		if _, ok := stats.FieldValueOccurences[f.Key]; ok {
			if _, ok := stats.FieldValueOccurences[f.Key][f.Value]; !ok {
				stats.FieldOccurences[f.Key]++
			}
			stats.FieldValueOccurences[f.Key][f.Value] += f.Occurrences
		} else {
			stats.FieldOccurences[f.Key]++
			stats.FieldValueOccurences[f.Key] = map[string]int{}
			stats.FieldValueOccurences[f.Key][f.Value] = f.Occurrences
		}
	}
	return nil
}

func (repo *inMemoryRepository) Get(id int64) (*Job, error) {
	if job, ok := repo.jobs[id]; !ok {
		return nil, errors.New("job with Id=" + string(id) + " not found")
	} else {
		return job, nil
	}
}

func (repo *inMemoryRepository) GetFieldOccurences(id int64) (map[string]int, error) {
	if stats, ok := repo.stats[id]; !ok {
		return nil, errors.New("job with Id=" + strconv.FormatInt(id, 10) + " not found")
	} else {
		repo.statMutexes[id].RLock()
		defer repo.statMutexes[id].RUnlock()
		copiedFieldOccurences := map[string]int{}
		for k, v := range stats.FieldOccurences {
			copiedFieldOccurences[k] = v
		}
		return copiedFieldOccurences, nil
	}
}

func (repo *inMemoryRepository) GetFieldValues(id int64, fieldName string) (map[string]int, error) {
	if stats, ok := repo.stats[id]; !ok {
		return nil, errors.New("job with ID=" + strconv.FormatInt(id, 10) + " not found")
	} else {
		repo.statMutexes[id].RLock()
		defer repo.statMutexes[id].RUnlock()
		values, ok := stats.FieldValueOccurences[fieldName]
		if !ok {
			return map[string]int{}, nil
		}
		return values, nil
	}
}

func (repo *inMemoryRepository) GetNumMatchedEvents(id int64) (int64, error) {
	if stats, ok := repo.stats[id]; !ok {
		return 0, errors.New("job with Id=" + strconv.FormatInt(id, 10) + " not found")
	} else {
		return stats.NumMatchedEvents, nil
	}
}

func (repo *inMemoryRepository) GetResults(id int64, skip int, take int) ([]int64, error) {
	if results, ok := repo.results[id]; !ok {
		return []int64{}, nil
	} else if results.Size() < skip {
		return nil, errors.New("out of bounds, there are fewer than skip=" + string(skip) + " elements in the job results (length=" + string(results.Size()) + ")")
	} else {
		ret := make([]int64, 0, take)
		it := results.Iterator()
		it.Next()
		for i := 0; i < skip; i++ {
			it.Next()
		}
		for i := 0; i < take; i++ {
			evt := it.Value().(events.EventIdAndTimestamp)
			ret = append(ret, evt.Id)
			if !it.Next() {
				break
			}
		}
		return ret, nil
	}
}

func (repo *inMemoryRepository) Insert(query string, startTime, endTime *time.Time) (*int64, error) {
	id := int64(len(repo.jobs))
	repo.jobs[id] = &Job{
		Id:        id,
		State:     JobStateRunning,
		Query:     query,
		StartTime: startTime,
		EndTime:   endTime,
	}
	repo.stats[id] = &JobStats{
		EstimatedProgress:    0,
		NumMatchedEvents:     0,
		FieldOccurences:      map[string]int{},
		FieldValueOccurences: map[string]map[string]int{},
	}
	repo.statMutexes[id] = &sync.RWMutex{}
	return &id, nil
}

func (repo *inMemoryRepository) UpdateState(id int64, state JobState) error {
	if _, ok := repo.jobs[id]; !ok {
		return errors.New("job with Id=" + string(id) + " not found")
	}
	repo.jobs[id].State = state
	return nil
}
