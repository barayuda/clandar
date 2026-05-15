package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register SQLite driver
)

// Store wraps a SQLite database connection and exposes it to the rest of the
// application.
type Store struct {
	DB *sql.DB
}

// Open opens (or creates) the SQLite database at dbPath, ensures the parent
// directory exists, and runs the embedded schema migration so all tables are
// present before the server starts accepting requests.
//
// schemaSQL is the full contents of db/schema.sql, passed in by the caller so
// this package does not need to know about the file layout.
func Open(dbPath, schemaSQL string) (*Store, error) {
	// Ensure the parent directory exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("store: create data directory %q: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}

	// SQLite works best with a single writer; limit the pool accordingly.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping database: %w", err)
	}

	// Enable WAL mode and foreign-key enforcement.
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: exec pragma %q: %w", p, err)
		}
	}

	// Apply schema migrations.
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}

	return &Store{DB: db}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}
