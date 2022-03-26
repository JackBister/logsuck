package pipeline

import (
	"path/filepath"

	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/indexedfiles"
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
			if m, err := filepath.Match(absGlob, evt.Source); err == nil && m {
				sourceToConfig[evt.Source] = &indexedFileConfigs[i]
				goto nextfile
			}
		}
	nextfile:
	}
	return sourceToConfig
}
