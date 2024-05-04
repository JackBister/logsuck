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

package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/util"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/dig"
)

type PostgresConfigRepository struct {
	pool *pgxpool.Pool

	broadcaster util.Broadcaster[struct{}]

	logger *slog.Logger
}

type PostgresConfigRepositoryParams struct {
	dig.In

	Ctx               context.Context
	Cfg               *config.Config
	Pool              *pgxpool.Pool
	ForceStaticConfig bool `name:"forceStaticConfig"`
	Logger            *slog.Logger
}

const createConfigUpdatedFunctionSql = `
CREATE OR REPLACE
FUNCTION config_updated_function() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
	PERFORM pg_notify('config_updated', '');
	RETURN NULL;
END;
$$;`

const createConfigUpdatedTriggerSql = `
CREATE OR REPLACE
TRIGGER config_updated_trigger AFTER
INSERT
OR UPDATE
OR DELETE
	ON
	Config EXECUTE FUNCTION config_updated_function();`

func NewPostgresConfigRepository(p PostgresConfigRepositoryParams) (config.Repository, error) {
	_, err := p.Pool.Exec(p.Ctx, "CREATE TABLE IF NOT EXISTS Config (id SERIAL NOT NULL PRIMARY KEY, config_json JSONB, modified TIMESTAMP)")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: %w", err)
	}
	_, err = p.Pool.Exec(p.Ctx, createConfigUpdatedFunctionSql)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to create config_updated_function: %w", err)
	}
	_, err = p.Pool.Exec(p.Ctx, createConfigUpdatedTriggerSql)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to create config_updated_trigger: %w", err)
	}
	r := p.Pool.QueryRow(p.Ctx, "SELECT COUNT(1) FROM Config")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to query for COUNT from Config: %w", err)
	}
	var c int
	err = r.Scan(&c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to scan COUNT from Config: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to close COUNT result: %w", err)
	}
	ret := &PostgresConfigRepository{pool: p.Pool, logger: p.Logger}
	if c == 0 && !p.ForceStaticConfig {
		err = ret.upsertInternal(p.Cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to upsert initial config: %w", err)
		}
	}
	conn, err := ret.pool.Acquire(p.Ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to acquire listen connection: %w", err)
	}
	_, err = conn.Exec(p.Ctx, "listen config_updated")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgresConfigRepository: failed to start listening for config_updated: %w", err)
	}
	go func(conn *pgxpool.Conn) {
		defer conn.Release()
		for {
			_, err := conn.Conn().WaitForNotification(p.Ctx)
			if err != nil {
				p.Logger.Error("Got error when waiting for config_updated notification. Will shut down listener.", slog.Any("error", err))
				return
			}
			ret.broadcaster.Broadcast(struct{}{})
		}
	}(conn)
	return ret, nil
}

func (s *PostgresConfigRepository) Changes() <-chan struct{} {
	return s.broadcaster.Subscribe()
}

func (s *PostgresConfigRepository) Get() (*config.ConfigResponse, error) {
	row := s.pool.QueryRow(context.Background(), "SELECT config_json, modified FROM Config ORDER BY modified DESC LIMIT 1")
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

func (s *PostgresConfigRepository) Upsert(c *config.Config) error {
	err := s.upsertInternal(c)
	if err != nil {
		return err
	}
	s.broadcaster.Broadcast(struct{}{})
	return nil
}

func (s *PostgresConfigRepository) upsertInternal(c *config.Config) error {
	jsonString, err := config.ToJSON(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	_, err = s.pool.Exec(context.Background(), "INSERT INTO Config (config_json, modified) VALUES ($1, $2)", string(jsonString), time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert new config into Config table: %w", err)
	}
	return nil
}
