package parser

import (
	"log"
	"regexp"
	"testing"
)

func TestExtract(t *testing.T) {
	jfp := JsonFileParser{Cfg: JsonParserConfig{
		EventDelimiter: regexp.MustCompile("\n"),
	}}

	res, _ := jfp.Extract(`{"level":"info","ts":1675014623.0479841,"logger":"SqliteEventsRepository","caller":"events/EventRepositorySqlite.go:146","msg":"added events","numEvents":118,"duration":"12.1532ms"}`)
	log.Println("res", res)
}
