package store

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS verdicts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		hype_score INTEGER NOT NULL,
		verdict_json TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) HasProcessed(url string) (bool, error) {
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM verdicts WHERE url = ? LIMIT 1", url).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SaveVerdict(url, title string, hypeScore int, verdictJSON string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO verdicts (url, title, hype_score, verdict_json) VALUES (?, ?, ?, ?)",
		url, title, hypeScore, verdictJSON,
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
