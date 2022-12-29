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
	cfg    *config.Config
	client http.Client

	changes            chan struct{}
	configPollInterval time.Duration
	cached             *config.ConfigResponse

	ticker *time.Ticker
}

func NewRemoteConfigSource(cfg *config.Config) config.ConfigSource {
	ret := RemoteConfigSource{
		cfg: cfg,
		client: http.Client{
			Timeout: cfg.ConfigPollInterval,
		},

		changes:            make(chan struct{}, 1), // We need to buffer to avoid hanging on startup
		configPollInterval: cfg.ConfigPollInterval,
		cached: &config.ConfigResponse{
			Modified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Cfg:      *cfg,
		},

		ticker: time.NewTicker(cfg.ConfigPollInterval),
	}
	ret.refresh()
	go func(r *RemoteConfigSource) {
		for {
			<-r.ticker.C
			oldModified := r.cached.Modified
			oldPollInterval := r.cached.Cfg.ConfigPollInterval
			r.refresh()
			newModified := r.cached.Modified
			newPollInterval := r.cached.Cfg.ConfigPollInterval
			if newModified != oldModified {
				r.changes <- struct{}{}
			}
			if newPollInterval != oldPollInterval {
				r.ticker.Stop()
				r.ticker = time.NewTicker(newPollInterval)
			}
		}
	}(&ret)
	return &ret
}

func (r *RemoteConfigSource) Changes() <-chan struct{} {
	return r.changes
}

func (r *RemoteConfigSource) Get() (*config.ConfigResponse, error) {
	return r.cached, nil
}

func (r *RemoteConfigSource) refresh() {
	now := time.Now()
	if r.cached == nil || now.Sub(r.cached.Modified) > 1*time.Minute {
		resp, err := r.client.Get(r.cfg.Forwarder.RecipientAddress + "/v1/config")
		if err != nil {
			log.Printf("got error when getting remote config: %v\n", err)
			return
		}
		defer resp.Body.Close()
		var cfgResp rpc.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&cfgResp)
		if err != nil {
			log.Printf("got error when getting remote config: error when decoding JSON: %v\n", err)
			return
		}

		cfg, err := config.FromJSON(cfgResp.Config)
		if err != nil {
			log.Printf("got error when getting remote config: error when converting config from JSON: %v\n", err)
			return
		}
		r.cached = &config.ConfigResponse{
			Modified: *cfgResp.Modified,
			Cfg:      *cfg,
		}
	}
}
