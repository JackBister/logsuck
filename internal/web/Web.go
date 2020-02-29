package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

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
		serialized, err := json.Marshal(results)
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

func (wi *webImpl) executeSearch(queryParams url.Values) ([]events.Event, *webError) {
	searchStrings, ok := queryParams["searchString"]
	if !ok || len(searchStrings) < 1 {
		return nil, &webError{
			err:  "searchString must be specified as a query parameter",
			code: 400,
		}
	}
	srch, err := search.Parse(strings.TrimSpace(searchStrings[0]))
	if err != nil {
		return nil, &webError{
			err:  "Got error when parsing search: " + err.Error(),
			code: 500,
		}
	}
	results := wi.eventRepo.Search(srch)
	return results, nil
}
