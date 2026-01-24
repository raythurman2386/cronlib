package sqlite

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements cronlib.JobStore using SQLite.
type Store struct {
	db *sql.DB
}

// New creates a new SQLite store.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS cron_job_state (
		id TEXT PRIMARY KEY,
		last_run DATETIME
	);
	CREATE TABLE IF NOT EXISTS cron_job_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT,
		start_time DATETIME,
		end_time DATETIME,
		success BOOLEAN,
		output TEXT
	);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *Store) GetLastRun(jobID string) (time.Time, error) {
	var t time.Time
	err := s.db.QueryRow("SELECT last_run FROM cron_job_state WHERE id = ?", jobID).Scan(&t)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	return t, err
}

func (s *Store) SetLastRun(jobID string, t time.Time) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO cron_job_state (id, last_run) VALUES (?, ?)", jobID, t)
	return err
}

func (s *Store) LogExecution(jobID string, start, end time.Time, success bool, out string) error {
	_, err := s.db.Exec("INSERT INTO cron_job_log (job_id, start_time, end_time, success, output) VALUES (?, ?, ?, ?, ?)",
		jobID, start, end, success, out)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
