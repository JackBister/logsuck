package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SqliteConfigRepository struct {
	db      *sql.DB
	changes chan struct{}
}

func NewSqliteConfigRepository(staticConfig *Config, db *sql.DB) (ConfigRepository, error) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Config (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, config_json TEXT, modified DATETIME)")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: %w", err)
	}
	r, err := db.Query("SELECT COUNT(1) FROM Config")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: failed to query for COUNT from Config: %w", err)
	}
	r.Next()
	var c int
	err = r.Scan(&c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: failed to scan COUNT from Config: %w", err)
	}
	err = r.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: failed to close COUNT result: %w", err)
	}
	changes := make(chan struct{}, 1) // We need to buffer to avoid hanging on startup
	ret := &SqliteConfigRepository{db: db, changes: changes}
	if c == 0 {
		err = ret.Upsert(staticConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: failed to upsert initial config: %w", err)
		}
	}
	return ret, nil
}

func (s *SqliteConfigRepository) Changes() <-chan struct{} {
	return s.changes
}

func (s *SqliteConfigRepository) Get() (*ConfigResponse, error) {
	row := s.db.QueryRow("SELECT config_json, modified FROM Config ORDER BY modified DESC LIMIT 1")
	if row == nil {
		return nil, fmt.Errorf("failed to get config_json row from Config table, got nil row")
	}
	var jsonString string
	var modified time.Time
	err := row.Scan(&jsonString, &modified)
	if err != nil {
		return nil, fmt.Errorf("got error when scanning config_json: %w", err)
	}
	var jsonCfg JsonConfig
	err = json.NewDecoder(strings.NewReader(jsonString)).Decode(&jsonCfg)
	if err != nil {
		return nil, fmt.Errorf("got error when decoding config_json: %w", err)
	}
	cfg, err := FromJSON(jsonCfg)
	if err != nil {
		return nil, fmt.Errorf("got error when converting JSON config: %w", err)
	}
	return &ConfigResponse{
		Cfg:      *cfg,
		Modified: modified,
	}, nil
}

func (s *SqliteConfigRepository) Upsert(c *Config) error {
	jsonString, err := ToJSON(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	b, err := json.Marshal(jsonString)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	_, err = s.db.Exec("INSERT INTO Config (config_json, modified) VALUES (?, ?)", string(b), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert new config into Config table: %w", err)
	}
	s.changes <- struct{}{}
	return nil
}
