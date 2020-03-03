package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/search"
)

type Web interface {
	Serve() error
}

type webImpl struct {
	cfg       *config.Config
	eventRepo events.Repository
}

type webError struct {
	err  string
	code int
}

func (w webError) Error() string {
	return w.err
}

func NewWeb(cfg *config.Config, eventRepo events.Repository) Web {
	return webImpl{
		cfg:       cfg,
		eventRepo: eventRepo,
	}
}

func (wi webImpl) Serve() error {
	http.Handle("/", http.FileServer(http.Dir("web/static")))

	http.HandleFunc("/api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		results, wErr := wi.executeSearch(queryParams)
		if wErr != nil {
			http.Error(w, wErr.err, wErr.code)
			return
		}
		res := SearchResult{
			Events:     results,
			FieldCount: aggregateFields(results),
		}
		serialized, err := json.Marshal(res)
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
		Addr: ":8080",
	}

	return s.ListenAndServe()
}

func (wi *webImpl) executeSearch(queryParams url.Values) ([]events.EventWithExtractedFields, *webError) {
	searchStrings, ok := queryParams["searchString"]
	if !ok || len(searchStrings) < 1 {
		return nil, &webError{
			err:  "searchString must be specified as a query parameter",
			code: 400,
		}
	}
	startTime, endTime, wErr := parseTimeParameters(queryParams)
	if wErr != nil {
		return nil, wErr
	}
	srch, err := search.Parse(strings.TrimSpace(searchStrings[0]), startTime, endTime)
	if err != nil {
		return nil, &webError{
			err:  "Got error when parsing search: " + err.Error(),
			code: 500,
		}
	}
	results := search.FilterEvents(wi.eventRepo, srch, wi.cfg)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	return results, nil
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

	if startTime == nil && endTime == nil {
		return nil, nil, &webError{
			err:  "One of relativeTime, startTime or endTime must be specified",
			code: 400,
		}
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

type SearchResult struct {
	Events     []events.EventWithExtractedFields
	FieldCount map[string]int
}
