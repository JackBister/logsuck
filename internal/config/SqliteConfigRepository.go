package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type SqliteConfigRepository struct {
	db *sql.DB
}

func NewSqliteConfigRepository(db *sql.DB) ConfigRepository {
	return &SqliteConfigRepository{db: db}
}

func (s *SqliteConfigRepository) SetAll(m map[string]string) error {
	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("got error when starting transaction to set config: %w", err)
	}
	for key, value := range m {
		log.Printf("setting config key=%v to value=%v\n", key, value)
		_, err := tx.Exec("INSERT INTO Config (key, value, modified) VALUES (?, ?, ?) ON CONFLICT (key) DO UPDATE SET value=excluded.value, modified=excluded.modified", key, value, now)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("got error when updating key=%v: %w", key, err)
		}
	}
	tx.Commit()
	return nil
}
