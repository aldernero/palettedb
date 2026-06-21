// Package palettedb provides a public API for opening and querying the
// palettedb SQLite database from external programs.
package palettedb

import (
	"fmt"

	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

// DB is an opened palette database.
type DB struct {
	inner *db.DB
}

// Open opens the palette database at path.
func Open(path string) (*DB, error) {
	inner, err := db.OpenAt(path)
	if err != nil {
		return nil, err
	}
	return &DB{inner: inner}, nil
}

// OpenDefault opens the palette database at the default XDG config path
// (~/.config/palettedb/palettedb.db).
func OpenDefault() (*DB, error) {
	inner, err := db.Open()
	if err != nil {
		return nil, err
	}
	return &DB{inner: inner}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.inner.Close()
}

// LoadSineByName loads a sine palette by name.
// Returns an error if the name does not exist or the entry is not a sine palette.
func (d *DB) LoadSineByName(name string) (gaul.SinePalette, error) {
	entry, err := d.inner.GetByName(name)
	if err != nil {
		return gaul.SinePalette{}, fmt.Errorf("palette %q not found: %w", name, err)
	}
	if entry.Type != "sine" {
		return gaul.SinePalette{}, fmt.Errorf("palette %q has type %q, want sine", name, entry.Type)
	}
	rec, err := d.inner.LoadSine(entry.ID)
	if err != nil {
		return gaul.SinePalette{}, err
	}
	return rec.Palette, nil
}
