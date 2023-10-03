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

package config

import "time"

type ConfigResponse struct {
	Modified time.Time
	Cfg      Config
}

type ConfigSource interface {
	Changes() <-chan struct{}
	Get() (*ConfigResponse, error)
}

type NullConfigSource struct {
}

func (n *NullConfigSource) Changes() <-chan struct{} {
	return make(<-chan struct{})
}

func (n *NullConfigSource) Get() (*ConfigResponse, error) {
	return &ConfigResponse{}, nil
}

type StaticConfigSource struct {
	Config Config
}

func (s *StaticConfigSource) Get() (*ConfigResponse, error) {
	return &ConfigResponse{Modified: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Cfg: s.Config}, nil
}

func (s *StaticConfigSource) Changes() <-chan struct{} {
	return make(<-chan struct{})
}
