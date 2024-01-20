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

package sqlite_config

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"

	"go.uber.org/dig"
)

type SqliteConfigRepository struct {
	db      *sql.DB
	changes chan struct{}

	logger *slog.Logger
}

type SqliteConfigRepositoryParams struct {
	dig.In

	Cfg               *config.Config
	Db                *sql.DB
	ForceStaticConfig bool `name:"forceStaticConfig"`
	Logger            *slog.Logger
}

func NewSqliteConfigRepository(p SqliteConfigRepositoryParams) (config.Repository, error) {
	_, err := p.Db.Exec("CREATE TABLE IF NOT EXISTS Config (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, config_json TEXT, modified DATETIME)")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: %w", err)
	}
	r, err := p.Db.Query("SELECT COUNT(1) FROM Config")
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
	changes := make(chan struct{})
	ret := &SqliteConfigRepository{db: p.Db, changes: changes, logger: p.Logger}
	if c == 0 && !p.ForceStaticConfig {
		err = ret.upsertInternal(p.Cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SqliteConfigRepository: failed to upsert initial config: %w", err)
		}
	}
	return ret, nil
}

func (s *SqliteConfigRepository) Changes() <-chan struct{} {
	return s.changes
}

func (s *SqliteConfigRepository) Get() (*config.ConfigResponse, error) {
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
	cfg, err := config.FromJSON([]byte(jsonString), s.logger)
	if err != nil {
		return nil, fmt.Errorf("got error when converting JSON config: %w", err)
	}
	return &config.ConfigResponse{
		Cfg:      *cfg,
		Modified: modified,
	}, nil
}

func (s *SqliteConfigRepository) Upsert(c *config.Config) error {
	err := s.upsertInternal(c)
	if err != nil {
		return err
	}
	s.changes <- struct{}{}
	return nil
}

func (s *SqliteConfigRepository) upsertInternal(c *config.Config) error {
	jsonString, err := config.ToJSON(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	_, err = s.db.Exec("INSERT INTO Config (config_json, modified) VALUES (?, ?)", string(jsonString), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert new config into Config table: %w", err)
	}
	return nil
}
