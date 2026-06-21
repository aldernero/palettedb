package db

import (
	"image/color"

	"github.com/aldernero/gaul"
)

// DiscretePaletteRecord wraps an ordered list of color stops with directory metadata.
type DiscretePaletteRecord struct {
	Entry
	Colors []color.Color
}

// Gradient reconstructs a gaul.Gradient from the stored color stops.
func (r *DiscretePaletteRecord) Gradient() gaul.Gradient {
	return gaul.NewGradientFromColors(r.Colors)
}

// SaveDiscrete inserts a directory entry and discrete color stops in one transaction.
func (d *DB) SaveDiscrete(name, description string, colors []color.Color) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO directory(name, type, description) VALUES (?,?,?)`,
		name, "discrete", description)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`INSERT INTO discrete_palettes(id) VALUES (?)`, id); err != nil {
		return 0, err
	}
	stmt, err := tx.Prepare(`INSERT INTO discrete_colors(palette_id, position, r, g, b) VALUES (?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for i, c := range colors {
		r64, g64, b64, _ := c.RGBA()
		if _, err := stmt.Exec(id, i, float64(r64)/65535.0, float64(g64)/65535.0, float64(b64)/65535.0); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// LoadDiscrete loads a DiscretePaletteRecord by directory ID.
func (d *DB) LoadDiscrete(id int64) (DiscretePaletteRecord, error) {
	entry, err := d.Get(id)
	if err != nil {
		return DiscretePaletteRecord{}, err
	}
	rows, err := d.Query(`SELECT r, g, b FROM discrete_colors WHERE palette_id=? ORDER BY position`, id)
	if err != nil {
		return DiscretePaletteRecord{}, err
	}
	defer rows.Close()
	var colors []color.Color
	for rows.Next() {
		var r, g, b float64
		if err := rows.Scan(&r, &g, &b); err != nil {
			return DiscretePaletteRecord{}, err
		}
		colors = append(colors, color.RGBA64{
			R: uint16(r * 65535),
			G: uint16(g * 65535),
			B: uint16(b * 65535),
			A: 65535,
		})
	}
	if err := rows.Err(); err != nil {
		return DiscretePaletteRecord{}, err
	}
	return DiscretePaletteRecord{Entry: entry, Colors: colors}, nil
}

// UpdateDiscrete replaces all color stops for an existing discrete palette.
func (d *DB) UpdateDiscrete(id int64, colors []color.Color) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM discrete_colors WHERE palette_id=?`, id); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO discrete_colors(palette_id, position, r, g, b) VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, c := range colors {
		r64, g64, b64, _ := c.RGBA()
		if _, err := stmt.Exec(id, i, float64(r64)/65535.0, float64(g64)/65535.0, float64(b64)/65535.0); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListDiscrete returns all directory entries of type "discrete" with their colors loaded.
func (d *DB) ListDiscrete() ([]DiscretePaletteRecord, error) {
	entries, err := d.ListAll()
	if err != nil {
		return nil, err
	}
	var records []DiscretePaletteRecord
	for _, e := range entries {
		if e.Type != "discrete" {
			continue
		}
		r, err := d.LoadDiscrete(e.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
