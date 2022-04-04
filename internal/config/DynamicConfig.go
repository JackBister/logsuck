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
	GetCurrentValueAsString(name string) string
	GetLastUpdateTime() *time.Time
	Ls(recursive bool) ([]string, bool)
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

func (d *RootDynamicConfig) GetCurrentValueAsString(name string) string {
	for _, cs := range d.configSources {
		v, ok := cs.Get(name)
		if ok {
			return v
		}
	}
	return ""
}

func (d *RootDynamicConfig) GetLastUpdateTime() *time.Time {
	var ret *time.Time
	for _, cs := range d.configSources {
		if p, ok := cs.(PollableConfigSource); ok {
			t, _ := p.GetLastUpdateTime()
			if t != nil && (ret == nil || t.After(*ret)) {
				ret = t
			}
		}
	}
	return ret
}

func (d *RootDynamicConfig) Ls(recursive bool) ([]string, bool) {
	ret := make([]string, 0)
	allOk := true
	for _, cs := range d.configSources {
		keys, ok := cs.GetKeys()
		if !ok {
			allOk = false
			continue
		}
		for _, k := range keys {
			if !recursive {
				k = strings.Split(k, ".")[0]
			}
			ret = append(ret, k)
		}
	}
	return ret, allOk
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

func (d *ChildDynamicConfig) GetCurrentValueAsString(name string) string {
	return d.parent.GetCurrentValueAsString(d.context + "." + name)
}

func (d *ChildDynamicConfig) GetLastUpdateTime() *time.Time {
	return d.GetLastUpdateTime()
}

func (d *ChildDynamicConfig) Ls(recursive bool) ([]string, bool) {
	keys, ok := d.parent.Ls(true)
	ret := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if strings.HasPrefix(k, d.context) {
			k = strings.TrimPrefix(k, d.context+".")
			if !recursive {
				k = strings.Split(k, ".")[0]
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			ret = append(ret, k)
		}
	}
	return ret, ok
}

type ConfigSource interface {
	Changes() <-chan struct{}
	Get(name string) (string, bool)
	GetKeys() ([]string, bool)
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
