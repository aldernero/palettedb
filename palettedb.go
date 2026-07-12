// Package palettedb provides a public API for opening and querying the
// palettedb SQLite database from external programs.
package palettedb

import (
	"fmt"

	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/builtins"
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

// LoadSineByName loads a sine palette by name, searching the user's saved
// palettes first and then the built-in palettes. Returns an error if no sine
// palette with that name exists in either.
func (d *DB) LoadSineByName(name string) (gaul.SinePalette, error) {
	if entry, err := d.inner.GetByName(name); err == nil && entry.Type == "sine" {
		rec, err := d.inner.LoadSine(entry.ID)
		if err != nil {
			return gaul.SinePalette{}, err
		}
		return rec.Palette, nil
	}
	if b, ok := builtins.ByName(name); ok {
		if sp, ok := b.SinePalette(); ok {
			return sp, nil
		}
	}
	return gaul.SinePalette{}, fmt.Errorf("sine palette %q not found", name)
}

// LoadDiscreteByName loads a discrete palette by name as a gaul.Gradient,
// searching the user's saved palettes first and then the built-in palettes.
// Returns an error if no discrete palette with that name exists in either.
func (d *DB) LoadDiscreteByName(name string) (gaul.Gradient, error) {
	if entry, err := d.inner.GetByName(name); err == nil && entry.Type == "discrete" {
		rec, err := d.inner.LoadDiscrete(entry.ID)
		if err != nil {
			return gaul.Gradient{}, err
		}
		return rec.Gradient(), nil
	}
	if b, ok := builtins.ByName(name); ok && b.Kind == "discrete" {
		return b.Gradient(), nil
	}
	return gaul.Gradient{}, fmt.Errorf("discrete palette %q not found", name)
}

// PaletteByName looks up a palette of either type by name (saved palettes first,
// then built-ins) and returns it as a gaul.Palette along with its type
// ("sine" or "discrete"). Use this when the caller does not know the type.
func (d *DB) PaletteByName(name string) (gaul.Palette, string, error) {
	if entry, err := d.inner.GetByName(name); err == nil {
		switch entry.Type {
		case "sine":
			rec, err := d.inner.LoadSine(entry.ID)
			if err != nil {
				return nil, "", err
			}
			sp := rec.Palette
			return &sp, "sine", nil
		case "discrete":
			rec, err := d.inner.LoadDiscrete(entry.ID)
			if err != nil {
				return nil, "", err
			}
			g := rec.Gradient()
			return &g, "discrete", nil
		}
	}
	if b, ok := builtins.ByName(name); ok {
		return b.Palette(), b.Kind, nil
	}
	return nil, "", fmt.Errorf("palette %q not found", name)
}
