// Copyright 2023 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package forwarder

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackbister/logsuck/internal/rpc"
	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/util"
	"go.uber.org/dig"
)

type RemoteConfigSource struct {
	cfg    *config.Config
	client http.Client

	cached *config.ConfigResponse

	ticker *time.Ticker

	broadcaster util.Broadcaster[struct{}]

	logger *slog.Logger
}

type RemoteConfigSourceParams struct {
	dig.In

	Cfg    *config.Config
	Logger *slog.Logger
}

func NewRemoteConfigSource(p RemoteConfigSourceParams) config.Source {
	ret := RemoteConfigSource{
		cfg: p.Cfg,
		client: http.Client{
			Timeout: p.Cfg.Forwarder.ConfigPollInterval,
		},

		cached: &config.ConfigResponse{
			Modified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Cfg:      *p.Cfg,
		},

		ticker: time.NewTicker(p.Cfg.Forwarder.ConfigPollInterval),

		logger: p.Logger,
	}
	ret.refresh()
	go func(r *RemoteConfigSource) {
		for {
			<-r.ticker.C
			oldModified := r.cached.Modified
			oldPollInterval := r.cached.Cfg.Forwarder.ConfigPollInterval
			r.refresh()
			newModified := r.cached.Modified
			newPollInterval := r.cached.Cfg.Forwarder.ConfigPollInterval
			if newModified != oldModified {
				r.broadcaster.Broadcast(struct{}{})
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
	return r.broadcaster.Subscribe()
}

func (r *RemoteConfigSource) Get() (*config.ConfigResponse, error) {
	return r.cached, nil
}

func (r *RemoteConfigSource) refresh() {
	now := time.Now()
	if r.cached == nil || now.Sub(r.cached.Modified) > 1*time.Minute {
		resp, err := r.client.Get(r.cfg.Forwarder.RecipientAddress + "/v1/config")
		if err != nil {
			r.logger.Error("got error when getting remote config",
				slog.Any("error", err))
			return
		}
		defer resp.Body.Close()
		var cfgResp rpc.ConfigResponse
		err = json.NewDecoder(resp.Body).Decode(&cfgResp)
		if err != nil {
			r.logger.Error("got error when getting remote config: error when decoding JSON",
				slog.Any("error", err))
			return
		}

		cfg, err := config.FromJSON(cfgResp.Config, r.logger)
		if err != nil {
			r.logger.Error("got error when getting remote config: error when converting config from JSON",
				slog.Any("error", err))
			return
		}
		r.cached = &config.ConfigResponse{
			Modified: *cfgResp.Modified,
			Cfg:      *cfg,
		}
	}
}
