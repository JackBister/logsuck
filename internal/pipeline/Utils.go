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

package pipeline

import (
	"path/filepath"

	"github.com/jackbister/logsuck/internal/indexedfiles"
	"github.com/jackbister/logsuck/pkg/logsuck/events"
)

func getSourceToIndexedFileConfig(evts []events.EventWithId, indexedFileConfigs []indexedfiles.IndexedFileConfig) map[string]*indexedfiles.IndexedFileConfig {
	sourceToConfig := map[string]*indexedfiles.IndexedFileConfig{}
	for _, evt := range evts {
		if _, ok := sourceToConfig[evt.Source]; ok {
			continue
		}
		for i, ifc := range indexedFileConfigs {
			absGlob, err := filepath.Abs(ifc.Filename)
			if err != nil {
				// TODO:
				continue
			}
			absSource, err := filepath.Abs(evt.Source)
			if err != nil {
				// TODO:
				continue
			}
			if m, err := filepath.Match(absGlob, absSource); err == nil && m {
				sourceToConfig[evt.Source] = &indexedFileConfigs[i]
				goto nextfile
			}
		}
	nextfile:
	}
	return sourceToConfig
}
