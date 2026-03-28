package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type ScoreEntry struct {
	Nick      string `json:"nick"`
	Score     int    `json:"score"`
	Days      int    `json:"days"`
	Timestamp string `json:"timestamp"`
}

type ScoreStore struct {
	db     *sql.DB
	mu     sync.Mutex
	lastIP map[string]time.Time
}

func NewScoreStore(dbPath string) *ScoreStore {
	if dbPath == "" {
		dbPath = "cats_scores.db"
	}
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0750); err != nil {
		fatal("failed to create scores db dir", "dir", dbDir, "err", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fatal("failed to open scores db", "path", dbPath, "err", err)
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS scores (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		nick      TEXT    NOT NULL,
		score     INTEGER NOT NULL,
		days      INTEGER NOT NULL,
		timestamp TEXT    NOT NULL
	)`)
	if err != nil {
		fatal("failed to create scores table", "path", dbPath, "err", err)
	}

	slog.Info("scores db ready", "path", dbPath)
	return &ScoreStore{db: db, lastIP: make(map[string]time.Time)}
}

func (s *ScoreStore) Top(n int) []ScoreEntry {
	rows, err := s.db.Query(
		`SELECT nick, score, days, timestamp
		 FROM scores
		 ORDER BY score DESC, days DESC, timestamp ASC
		 LIMIT ?`, n,
	)
	if err != nil {
		slog.Error("scores query failed", "err", err)
		return nil
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("scores close rows failed", "err", err)
		}
	}()

	var entries []ScoreEntry
	for rows.Next() {
		var e ScoreEntry
		if err := rows.Scan(&e.Nick, &e.Score, &e.Days, &e.Timestamp); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		slog.Error("scores rows iteration failed", "err", err)
		return nil
	}
	return entries
}

func (s *ScoreStore) Add(entry ScoreEntry, ip string) (string, int) {
	s.mu.Lock()
	last, ok := s.lastIP[ip]
	if ok && time.Since(last) < 5*time.Second {
		s.mu.Unlock()
		return "too many requests", http.StatusTooManyRequests
	}
	s.lastIP[ip] = time.Now()
	s.mu.Unlock()

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO scores (nick, score, days, timestamp) VALUES (?, ?, ?, ?)`,
		entry.Nick, entry.Score, entry.Days, entry.Timestamp,
	)
	if err != nil {
		slog.Error("scores insert failed", "err", err, "nick", entry.Nick, "score", entry.Score, "days", entry.Days)
		return "db error", http.StatusInternalServerError
	}
	return "", 0
}
