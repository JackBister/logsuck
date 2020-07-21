package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/jobs"
	"github.com/markbates/pkger"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/parser"
)

type Web interface {
	Serve() error
}

type webImpl struct {
	cfg       *config.Config
	eventRepo events.Repository
	jobRepo   jobs.Repository
	jobEngine *jobs.Engine
}

type webError struct {
	err  string
	code int
}

func (w webError) Error() string {
	return w.err
}

func NewWeb(cfg *config.Config, eventRepo events.Repository, jobRepo jobs.Repository, jobEngine *jobs.Engine) Web {
	return webImpl{
		cfg:       cfg,
		eventRepo: eventRepo,
		jobRepo:   jobRepo,
		jobEngine: jobEngine,
	}
}

func (wi webImpl) Serve() error {
	if wi.cfg.Web.UsePackagedFiles {
		http.Handle("/", http.FileServer(pkger.Dir("/web/static/dist")))
	} else {
		http.Handle("/", http.FileServer(http.Dir("web/static/dist")))
	}

	http.HandleFunc("/api/v1/startJob", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		searchStrings, ok := queryParams["searchString"]
		if !ok || len(searchStrings) < 1 {
			http.Error(w, "searchString must be specified as a query parameter", 400)
			return
		}
		startTime, endTime, wErr := parseTimeParameters(queryParams)
		if wErr != nil {
			http.Error(w, wErr.err, wErr.code)
			return
		}
		id, err := wi.jobEngine.StartJob(strings.TrimSpace(searchStrings[0]), startTime, endTime)
		if err != nil {
			http.Error(w, "Got error when creating job: "+err.Error(), 500)
			return
		}
		serialized, err := json.Marshal(id)
		if err != nil {
			http.Error(w, "Got error when serializing id:"+err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(serialized)
		if err != nil {
			http.Error(w, "Got error when writing results:"+err.Error(), 500)
			return
		}
	})

	http.HandleFunc("/api/v1/abortJob", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		jobIdString, ok := queryParams["jobId"]
		if !ok || len(jobIdString) < 1 {
			http.Error(w, "jobId must be specified as a query parameter", 400)
			return
		}
		jobId, err := strconv.ParseInt(jobIdString[0], 10, 64)
		if err != nil {
			http.Error(w, "jobId must be an integer", 400)
			return
		}
		err = wi.jobEngine.Abort(jobId)
		if err != nil {
			http.Error(w, "failed to abort job: "+err.Error(), 500)
			return
		}
	})

	http.HandleFunc("/api/v1/jobStats", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		jobIdString, ok := queryParams["jobId"]
		if !ok || len(jobIdString) < 1 {
			http.Error(w, "jobId must be specified as a query parameter", 400)
			return
		}
		jobId, err := strconv.ParseInt(jobIdString[0], 10, 64)
		if err != nil {
			http.Error(w, "jobId must be an integer", 400)
			return
		}
		job, err := wi.jobRepo.Get(jobId)
		if err != nil {
			http.Error(w, "got error when retrieving job: "+err.Error(), 500)
			return
		}
		fieldCount, err := wi.jobRepo.GetFieldOccurences(jobId)
		if err != nil {
			http.Error(w, "Got error when getting field occurences:"+err.Error(), 500)
			return
		}
		numMatched, err := wi.jobRepo.GetNumMatchedEvents(jobId)
		if err != nil {
			http.Error(w, "Got error when getting number of matched events:"+err.Error(), 500)
			return
		}
		ret := PollResult{
			State:            job.State,
			FieldCount:       fieldCount,
			NumMatchedEvents: numMatched,
		}
		serialized, err := json.Marshal(ret)
		if err != nil {
			http.Error(w, "Got error when serializing results:"+err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(serialized)
		if err != nil {
			http.Error(w, "Got error when writing results:"+err.Error(), 500)
			return
		}
	})

	http.HandleFunc("/api/v1/jobResults", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		jobIdString, ok := queryParams["jobId"]
		if !ok || len(jobIdString) < 1 {
			http.Error(w, "jobId must be specified as a query parameter", 400)
			return
		}
		jobId, err := strconv.ParseInt(jobIdString[0], 10, 64)
		if err != nil {
			http.Error(w, "jobId must be an integer", 400)
			return
		}
		skipString, ok := queryParams["skip"]
		if !ok || len(skipString) < 1 {
			http.Error(w, "skip must be specified as a query parameter", 400)
			return
		}
		skip, err := strconv.Atoi(skipString[0])
		if err != nil {
			http.Error(w, "skip must be an integer", 400)
			return
		}
		takeString, ok := queryParams["take"]
		if !ok || len(takeString) < 1 {
			http.Error(w, "take must be specified as a query parameter", 400)
			return
		}
		take, err := strconv.Atoi(takeString[0])
		if err != nil {
			http.Error(w, "take must be an integer", 400)
			return
		}
		eventIds, err := wi.jobRepo.GetResults(jobId, skip, take)
		if err != nil {
			http.Error(w, "got error when getting eventIds, err="+err.Error(), 500)
			return
		}
		results, err := wi.eventRepo.GetByIds(eventIds)
		if err != nil {
			http.Error(w, "got error when getting events, err="+err.Error(), 500)
			return
		}

		retResults := make([]events.EventWithExtractedFields, 0, len(results))
		for _, r := range results {
			fields := parser.ExtractFields(r.Raw, wi.cfg.FieldExtractors)
			retResults = append(retResults, events.EventWithExtractedFields{
				Id:        r.Id,
				Raw:       r.Raw,
				Source:    r.Source,
				Timestamp: r.Timestamp,
				Fields:    fields,
			})
		}

		serialized, err := json.Marshal(retResults)
		if err != nil {
			http.Error(w, "Got error when serializing results:"+err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(serialized)
		if err != nil {
			http.Error(w, "Got error when writing results:"+err.Error(), 500)
		}
	})

	http.HandleFunc("/api/v1/jobFieldStats", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		jobIdString, ok := queryParams["jobId"]
		if !ok || len(jobIdString) < 1 {
			http.Error(w, "jobId must be specified as a query parameter", 400)
			return
		}
		jobId, err := strconv.ParseInt(jobIdString[0], 10, 64)
		if err != nil {
			http.Error(w, "jobId must be an integer", 400)
			return
		}
		fieldName, ok := queryParams["fieldName"]
		if !ok || len(fieldName) < 1 {
			http.Error(w, "fieldName must be specified as a query parameter", 400)
			return
		}
		values, err := wi.jobRepo.GetFieldValues(jobId, fieldName[0])
		if err != nil {
			http.Error(w, "Got error when getting field values:"+err.Error(), 500)
		}
		serialized, err := json.Marshal(values)
		if err != nil {
			http.Error(w, "Got error when serializing results:"+err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(serialized)
		if err != nil {
			http.Error(w, "Got error when writing results:"+err.Error(), 500)
		}
	})

	s := http.Server{
		Addr: wi.cfg.Web.Address,
	}

	return s.ListenAndServe()
}

func parseTimeParameters(queryParams url.Values) (*time.Time, *time.Time, *webError) {
	relativeTimes, hasRelativeTimes := queryParams["relativeTime"]
	absoluteStarts, hasAbsoluteStarts := queryParams["startTime"]
	absoluteEnds, hasAbsoluteEnds := queryParams["endTime"]

	if hasRelativeTimes && len(relativeTimes) > 0 {
		relative, err := time.ParseDuration(relativeTimes[0])
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing relativeTime: " + err.Error(),
				code: 400,
			}
		}
		startTime := time.Now().Add(relative)
		return &startTime, nil, nil
	}
	var startTime *time.Time
	var endTime *time.Time
	if hasAbsoluteStarts && len(absoluteStarts) > 0 {
		t, err := time.Parse(time.RFC3339, absoluteStarts[0])
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing startTime: " + err.Error(),
				code: 400,
			}
		}
		startTime = &t
	}
	if hasAbsoluteEnds && len(absoluteEnds) > 0 {
		t, err := time.Parse(time.RFC3339, absoluteEnds[0])
		if err != nil {
			return nil, nil, &webError{
				err:  "Got error when parsing endTime: " + err.Error(),
				code: 400,
			}
		}
		endTime = &t
	}

	return startTime, endTime, nil
}

func aggregateFields(inputEvents []events.EventWithExtractedFields) map[string]int {
	ret := map[string]int{}
	for _, evt := range inputEvents {
		for field := range evt.Fields {
			if i, ok := ret[field]; ok {
				ret[field] = i + 1
			} else {
				ret[field] = 1
			}
		}
	}
	return ret
}

type PollResult struct {
	State            jobs.JobState
	FieldCount       map[string]int
	NumMatchedEvents int64
}

type SearchResult struct {
	Events     []events.EventWithExtractedFields
	FieldCount map[string]int
}
