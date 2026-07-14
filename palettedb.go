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

// DefaultPath returns the default database location used by OpenDefault
// (~/.config/palettedb/palettedb.db, honoring XDG_CONFIG_HOME).
func DefaultPath() (string, error) {
	return db.DefaultPath()
}

// Entry describes a palette stored in the database.
type Entry struct {
	Name        string
	Type        string // "sine" or "discrete"
	Description string
}

// List returns all palettes stored in the database ordered by name.
// Built-in palettes are not included.
func (d *DB) List() ([]Entry, error) {
	rows, err := d.inner.ListAll()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, Entry{Name: r.Name, Type: r.Type, Description: r.Description})
	}
	return entries, nil
}

// ListNames returns the names of stored palettes of the given type
// ("sine" or "discrete") ordered by name. Built-in palettes are not included.
func (d *DB) ListNames(paletteType string) ([]string, error) {
	entries, err := d.List()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Type == paletteType {
			names = append(names, e.Name)
		}
	}
	return names, nil
}

// Builtins returns the read-only palettes compiled into the library (discrete
// colormaps such as viridis and turbo, plus sine palettes) ordered by name.
// They resolve through the same LoadSineByName/LoadDiscreteByName/PaletteByName
// lookups as stored palettes.
func Builtins() []Entry {
	all := builtins.All()
	entries := make([]Entry, 0, len(all))
	for _, b := range all {
		entries = append(entries, Entry{Name: b.Name, Type: b.Kind, Description: b.Description})
	}
	return entries
}

// BuiltinNames returns the names of built-in palettes of the given type
// ("sine" or "discrete") ordered by name.
func BuiltinNames(paletteType string) []string {
	var names []string
	for _, e := range Builtins() {
		if e.Type == paletteType {
			names = append(names, e.Name)
		}
	}
	return names
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
