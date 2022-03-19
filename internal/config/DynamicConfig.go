// Copyright 2022 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DynamicArrayProperty struct {
	name         string
	defaultValue []interface{}
	cfg          *RootDynamicConfig
}

func (d DynamicArrayProperty) Get() ([]interface{}, bool) {
	for _, cs := range d.cfg.configSources {
		val, ok := cs.Get(d.name)
		if !ok {
			continue
		}
		var arr []interface{}
		err := json.NewDecoder(strings.NewReader(val)).Decode(&arr)
		if err != nil {
			continue
		}
		return arr, true
	}
	return d.defaultValue, false
}

type DynamicIntProperty struct {
	name         string
	defaultValue int
	cfg          *RootDynamicConfig
}

func (d DynamicIntProperty) Get() (int, bool) {
	for _, cs := range d.cfg.configSources {
		val, ok := cs.Get(d.name)
		if !ok {
			continue
		}
		i, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		return i, true
	}
	return d.defaultValue, false
}

type DynamicStringProperty struct {
	name         string
	defaultValue string
	cfg          *RootDynamicConfig
}

func (d DynamicStringProperty) Get() (string, bool) {
	for _, cs := range d.cfg.configSources {
		val, ok := cs.Get(d.name)
		if !ok {
			continue
		}
		return val, true
	}
	return d.defaultValue, false
}

type DynamicConfig interface {
	Cd(name string) DynamicConfig
	Changes() <-chan struct{}
	GetArray(name string, defaultValue []interface{}) DynamicArrayProperty
	GetInt(name string, defaultValue int) DynamicIntProperty
	GetString(name string, defaultValue string) DynamicStringProperty
}

type RootDynamicConfig struct {
	configSources []ConfigSource
	changes       <-chan struct{}
}

func NewDynamicConfig(configSources []ConfigSource) DynamicConfig {
	cs := make([]<-chan struct{}, len(configSources))
	for i := range configSources {
		cs[i] = configSources[i].Changes()
	}
	return &RootDynamicConfig{
		configSources: configSources,
		changes:       mergeChannels(cs),
	}
}

func (d *RootDynamicConfig) Cd(name string) DynamicConfig {
	return &ChildDynamicConfig{
		context: name,
		parent:  d,
	}
}

func (d *RootDynamicConfig) Changes() <-chan struct{} {
	return d.changes
}

func (d *RootDynamicConfig) GetArray(name string, defaultValue []interface{}) DynamicArrayProperty {
	return DynamicArrayProperty{name: name, defaultValue: defaultValue, cfg: d}
}

func (d *RootDynamicConfig) GetInt(name string, defaultValue int) DynamicIntProperty {
	return DynamicIntProperty{name: name, defaultValue: defaultValue, cfg: d}
}

func (d *RootDynamicConfig) GetString(name string, defaultValue string) DynamicStringProperty {
	return DynamicStringProperty{name: name, defaultValue: defaultValue, cfg: d}
}

type ChildDynamicConfig struct {
	context string
	parent  *RootDynamicConfig
}

func (d *ChildDynamicConfig) Cd(name string) DynamicConfig {
	return &ChildDynamicConfig{
		context: d.context + "." + name,
		parent:  d.parent,
	}
}

func (d *ChildDynamicConfig) Changes() <-chan struct{} {
	return d.parent.changes
}

func (d *ChildDynamicConfig) GetArray(name string, defaultValue []interface{}) DynamicArrayProperty {
	return DynamicArrayProperty{name: d.context + "." + name, defaultValue: defaultValue, cfg: d.parent}
}

func (d *ChildDynamicConfig) GetInt(name string, defaultValue int) DynamicIntProperty {
	return DynamicIntProperty{name: d.context + "." + name, defaultValue: defaultValue, cfg: d.parent}
}

func (d *ChildDynamicConfig) GetString(name string, defaultValue string) DynamicStringProperty {
	return DynamicStringProperty{name: d.context + "." + name, defaultValue: defaultValue, cfg: d.parent}
}

type ConfigSource interface {
	Changes() <-chan struct{}
	Get(name string) (string, bool)
}

type PollableConfigSource interface {
	ConfigSource
	GetLastUpdateTime() (*time.Time, error)
}

func mergeChannels(cs []<-chan struct{}) <-chan struct{} {
	ret := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan struct{}) {
			for s := range c {
				ret <- s
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(ret)
	}()
	return ret
}
