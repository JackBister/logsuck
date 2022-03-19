package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type SqliteConfigSource struct {
	db      *sql.DB
	changes chan struct{}
}

func NewSqliteConfigSource(db *sql.DB) (*SqliteConfigSource, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Config (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL, value TEXT NOT NULL, modified DATETIME NOT NULL, UNIQUE(key))")
	if err != nil {
		return nil, fmt.Errorf("error creating config table: %w", err)
	}

	return &SqliteConfigSource{
		db:      db,
		changes: make(chan struct{}),
	}, nil
}

func (s *SqliteConfigSource) Changes() <-chan struct{} {
	return s.changes
}

func (s *SqliteConfigSource) Get(name string) (string, bool) {
	rows, err := s.db.Query("SELECT value FROM Config WHERE key=?", name)
	if err != nil {
		log.Printf("got error when getting config key='%s' from sqlite: %v\n", name, err)
		return "", false
	}
	if !rows.Next() {
		return "", false
	}
	var ret string
	err = rows.Scan(&ret)
	if err != nil {
		log.Printf("got error when scanning config key='%s' from sqlite: %v\n", name, err)
		return "", false
	}
	return ret, true
}

func (s *SqliteConfigSource) GetLastUpdateTime() (*time.Time, error) {
	rows, err := s.db.Query("SELECT MAX(modified) FROM Config")
	if err != nil {
		return nil, fmt.Errorf("got error when getting last update time: %w", err)
	}
	if !rows.Next() {
		return nil, fmt.Errorf("did not get any rows when getting last update time")
	}
	var ret *time.Time
	err = rows.Scan(&ret)
	if err != nil {
		return nil, fmt.Errorf("got error when scanning last update time: %w", err)
	}
	return ret, nil
}
