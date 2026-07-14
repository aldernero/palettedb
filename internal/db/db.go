package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS directory (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    type        TEXT    NOT NULL CHECK(type IN ('sine','discrete')),
    description TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sine_palettes (
    id    INTEGER PRIMARY KEY REFERENCES directory(id) ON DELETE CASCADE,
    ax REAL NOT NULL, ay REAL NOT NULL, az REAL NOT NULL,
    bx REAL NOT NULL, by REAL NOT NULL, bz REAL NOT NULL,
    cx REAL NOT NULL, cy REAL NOT NULL, cz REAL NOT NULL,
    dx REAL NOT NULL, dy REAL NOT NULL, dz REAL NOT NULL,
    alpha REAL NOT NULL DEFAULT 1.0,
    color_space TEXT NOT NULL DEFAULT 'RGB'
);

CREATE TABLE IF NOT EXISTS discrete_palettes (
    id INTEGER PRIMARY KEY REFERENCES directory(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS discrete_colors (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    palette_id INTEGER NOT NULL REFERENCES discrete_palettes(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    pos        REAL,
    r REAL NOT NULL, g REAL NOT NULL, b REAL NOT NULL,
    UNIQUE(palette_id, position)
);

CREATE INDEX IF NOT EXISTS idx_dc_palette ON discrete_colors(palette_id, position);
`

// DB wraps *sql.DB with application-level helpers.
type DB struct {
	*sql.DB
}

// migrate applies schema changes to existing databases (idempotent).
func migrate(sqldb *sql.DB) error {
	stmts := []string{
		`ALTER TABLE sine_palettes ADD COLUMN color_space TEXT NOT NULL DEFAULT 'RGB'`,
		// Normalized stop position in [0,1]. Nullable: legacy rows have NULL and
		// are treated as evenly spaced at load time.
		`ALTER TABLE discrete_colors ADD COLUMN pos REAL`,
	}
	for _, s := range stmts {
		if _, err := sqldb.Exec(s); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

// Open opens (or creates) the SQLite database at the XDG config path.
func Open() (*DB, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "palettedb.db")
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	if _, err := sqldb.Exec(schema); err != nil {
		sqldb.Close()
		return nil, err
	}
	if err := migrate(sqldb); err != nil {
		sqldb.Close()
		return nil, err
	}
	return &DB{sqldb}, nil
}

// OpenAt opens (or creates) a database at an explicit path, used in tests.
func OpenAt(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	if _, err := sqldb.Exec(schema); err != nil {
		sqldb.Close()
		return nil, err
	}
	if err := migrate(sqldb); err != nil {
		sqldb.Close()
		return nil, err
	}
	return &DB{sqldb}, nil
}

// DefaultPath returns the database path used by Open
// (~/.config/palettedb/palettedb.db, honoring XDG_CONFIG_HOME).
func DefaultPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "palettedb.db"), nil
}

func configDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "palettedb"), nil
}
