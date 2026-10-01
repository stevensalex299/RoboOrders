package store

import (
	"context"
	"database/sql"
	"strconv"
)

func (s *Store) GetIngestInt(ctx context.Context, key string) (int, error) {
	var val string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM ingest_state WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) SetIngestInt(ctx context.Context, key string, n int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ingest_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, strconv.Itoa(n))
	return err
}
