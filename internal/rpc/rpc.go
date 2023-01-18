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

package rpc

import (
	"time"

	"github.com/jackbister/logsuck/internal/config"
)

type RawEvent struct {
	Raw      string
	Host     string
	Source   string
	SourceId string
	Offset   int64
}

type ReceiveEventsRequest struct {
	HostType string
	Events   []RawEvent
}

type ConfigResponse struct {
	Modified *time.Time
	Config   config.JsonConfig
}
