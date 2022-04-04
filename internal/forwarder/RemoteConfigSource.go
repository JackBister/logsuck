package forwarder

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackbister/logsuck/internal/config"
	"github.com/jackbister/logsuck/internal/rpc"
)

type RemoteConfigSource struct {
	cfg    *config.StaticConfig
	client http.Client

	changes     chan struct{}
	cached      *config.MapConfigSource
	lastUpdated *time.Time
}

func NewRemoteConfigSource(cfg *config.StaticConfig) config.ConfigSource {
	return &RemoteConfigSource{
		cfg: cfg,
		client: http.Client{
			Timeout: 30 * time.Second,
		},

		changes: make(chan struct{}),
	}
}

func (r *RemoteConfigSource) Changes() <-chan struct{} {
	return r.changes
}

func (r *RemoteConfigSource) Get(name string) (string, bool) {
	r.refresh()
	return r.cached.Get(name)
}

func (r *RemoteConfigSource) GetKeys() ([]string, bool) {
	r.refresh()
	return r.cached.GetKeys()
}

func (r *RemoteConfigSource) GetLastUpdateTime() (*time.Time, error) {
	return r.lastUpdated, nil
}

func (r *RemoteConfigSource) refresh() {
	defer func() {
		if r.cached == nil {
			r.cached = config.NewMapConfigSource(map[string]any{})
		}
	}()
	now := time.Now()
	if r.cached == nil || r.lastUpdated == nil || now.Sub(*r.lastUpdated) > 1*time.Minute {
		resp, err := r.client.Get(r.cfg.Forwarder.RecipientAddress + "/v1/config")
		if err != nil {
			log.Printf("got error when getting remote config: %v\n", err)
			return
		}
		defer resp.Body.Close()
		var cfg rpc.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&cfg)
		if err != nil {
			log.Printf("got error when decoding remote config: %v\n", err)
			return
		}
		m := make(map[string]any, len(cfg.Config))
		for k, v := range cfg.Config {
			m[k] = v
		}
		r.cached = config.NewMapConfigSource(m)
		r.lastUpdated = cfg.LastUpdateTime
	}
}
