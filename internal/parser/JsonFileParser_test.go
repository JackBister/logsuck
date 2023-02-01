package parser

import (
	"regexp"
	"testing"

	"go.uber.org/zap"
)

func TestJsonFileParserExtract(t *testing.T) {
	l, _ := zap.NewDevelopment()
	p := JsonFileParser{
		Cfg: JsonParserConfig{
			EventDelimiter: regexp.MustCompile("\n"),
		},
		Logger: l,
	}

	r, _ := p.Extract(`
{"level":"info","ts":1675006830.0893068,"logger":"reloadFileWatchers","caller":"logsuck/main.go:339","msg":"reloading file watchers","newIndexedFilesLen":3,"oldIndexedFilesLen":0}
	`)
	if r.Fields["level"] != "info" {
		t.FailNow()
	}

}
