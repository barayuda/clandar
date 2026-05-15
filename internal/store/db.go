package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/barayuda/clandar/internal/config"

	// Local SQLite driver (pure Go, no CGo) — used for local dev.
	_ "modernc.org/sqlite"

	// Turso libSQL driver — used for cloud deployment.
	// The blank import registers the "libsql" driver with database/sql.
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// Store wraps a database connection and exposes it to the rest of the app.
type Store struct {
	DB *sql.DB
}

// Open opens the database using the active configuration:
//   - If TURSO_DATABASE_URL is set, connects to Turso cloud via libSQL.
//   - Otherwise, opens (or creates) a local SQLite file at cfg.DBPath.
//
// schemaSQL is the full contents of db/schema.sql, executed on every open
// so all tables exist before the server starts accepting requests.
func Open(cfg *config.Config, schemaSQL string) (*Store, error) {
	var (
		db     *sql.DB
		err    error
		driver string
		dsn    string
	)

	if cfg.IsRemoteDB() {
		// ── Turso cloud ───────────────────────────────────────────────────
		driver = "libsql"
		dsn = cfg.TursoDatabaseURL + "?authToken=" + cfg.TursoAuthToken
	} else {
		// ── Local SQLite file ─────────────────────────────────────────────
		driver = "sqlite"
		dsn = cfg.DBPath

		// Ensure the parent directory exists.
		dir := filepath.Dir(cfg.DBPath)
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create data directory %q: %w", dir, err)
		}
	}

	db, err = sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open database (%s): %w", driver, err)
	}

	// SQLite (local and Turso) works best with a single writer.
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping database: %w", err)
	}

	// Enable WAL mode and foreign-key enforcement for local SQLite only.
	// Turso manages these settings server-side.
	if !cfg.IsRemoteDB() {
		pragmas := []string{
			"PRAGMA journal_mode=WAL;",
			"PRAGMA foreign_keys=ON;",
		}
		for _, p := range pragmas {
			if _, err = db.Exec(p); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("store: exec pragma %q: %w", p, err)
			}
		}
	}

	// Apply schema (CREATE TABLE IF NOT EXISTS — idempotent).
	if _, err = db.Exec(schemaSQL); err != nil {
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
