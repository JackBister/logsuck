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

package parser

import (
	"log/slog"
	"regexp"
	"testing"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
)

func TestJsonFileParserExtract(t *testing.T) {
	p := JsonFileParser{
		Cfg: config.JsonParserConfig{
			EventDelimiter: regexp.MustCompile("\n"),
		},
		Logger: slog.Default(),
	}

	r, _ := p.Extract(`
{"level":"info","ts":1675006830.0893068,"logger":"reloadFileWatchers","caller":"logsuck/main.go:339","msg":"reloading file watchers","newIndexedFilesLen":3,"oldIndexedFilesLen":0}
	`)
	if r.Fields["level"] != "info" {
		t.FailNow()
	}

}
