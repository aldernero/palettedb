package db

import (
	"database/sql"
	"image/color"

	"github.com/aldernero/gaul"
)

// ColorStop is a single color stop with a normalized position in [0,1].
type ColorStop struct {
	Color color.Color
	Pos   float64
}

// DiscretePaletteRecord wraps an ordered list of color stops with directory metadata.
type DiscretePaletteRecord struct {
	Entry
	Stops []ColorStop
}

// Colors returns just the stop colors, in order.
func (r *DiscretePaletteRecord) Colors() []color.Color {
	cs := make([]color.Color, len(r.Stops))
	for i, s := range r.Stops {
		cs[i] = s.Color
	}
	return cs
}

// Positions returns just the stop positions, in order.
func (r *DiscretePaletteRecord) Positions() []float64 {
	ps := make([]float64, len(r.Stops))
	for i, s := range r.Stops {
		ps[i] = s.Pos
	}
	return ps
}

// Gradient reconstructs a position-aware gaul.Gradient from the stored stops.
func (r *DiscretePaletteRecord) Gradient() gaul.Gradient {
	return gaul.NewGradientFromColorStops(r.Colors(), r.Positions())
}

// SaveDiscrete inserts a directory entry and discrete color stops in one transaction.
func (d *DB) SaveDiscrete(name, description string, stops []ColorStop) (int64, error) {
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
	if err := insertStops(tx, id, stops); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// LoadDiscrete loads a DiscretePaletteRecord by directory ID.
func (d *DB) LoadDiscrete(id int64) (DiscretePaletteRecord, error) {
	entry, err := d.Get(id)
	if err != nil {
		return DiscretePaletteRecord{}, err
	}
	rows, err := d.Query(`SELECT r, g, b, pos FROM discrete_colors WHERE palette_id=? ORDER BY position`, id)
	if err != nil {
		return DiscretePaletteRecord{}, err
	}
	defer rows.Close()
	var stops []ColorStop
	var valid []bool
	for rows.Next() {
		var r, g, b float64
		var pos sql.NullFloat64
		if err := rows.Scan(&r, &g, &b, &pos); err != nil {
			return DiscretePaletteRecord{}, err
		}
		stops = append(stops, ColorStop{
			Color: color.RGBA64{
				R: uint16(r * 65535),
				G: uint16(g * 65535),
				B: uint16(b * 65535),
				A: 65535,
			},
			Pos: pos.Float64,
		})
		valid = append(valid, pos.Valid)
	}
	if err := rows.Err(); err != nil {
		return DiscretePaletteRecord{}, err
	}
	// Legacy rows have NULL pos; treat them as evenly spaced.
	backfillEvenPositions(stops, valid)
	return DiscretePaletteRecord{Entry: entry, Stops: stops}, nil
}

// UpdateDiscrete replaces all color stops for an existing discrete palette.
func (d *DB) UpdateDiscrete(id int64, stops []ColorStop) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM discrete_colors WHERE palette_id=?`, id); err != nil {
		return err
	}
	if err := insertStops(tx, id, stops); err != nil {
		return err
	}
	return tx.Commit()
}

// ListDiscrete returns all directory entries of type "discrete" with their stops loaded.
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

// insertStops writes the given stops into discrete_colors for palette id.
func insertStops(tx *sql.Tx, id int64, stops []ColorStop) error {
	stmt, err := tx.Prepare(`INSERT INTO discrete_colors(palette_id, position, pos, r, g, b) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, s := range stops {
		r64, g64, b64, _ := s.Color.RGBA()
		if _, err := stmt.Exec(id, i, s.Pos,
			float64(r64)/65535.0, float64(g64)/65535.0, float64(b64)/65535.0); err != nil {
			return err
		}
	}
	return nil
}

// backfillEvenPositions fills in evenly spaced positions for stops whose stored
// position was NULL (legacy rows saved before the pos column existed). valid[i]
// reports whether stop i had a non-NULL stored position.
func backfillEvenPositions(stops []ColorStop, valid []bool) {
	n := len(stops)
	for i := range stops {
		if i < len(valid) && valid[i] {
			continue
		}
		if n <= 1 {
			stops[i].Pos = 0
		} else {
			stops[i].Pos = float64(i) / float64(n-1)
		}
	}
}
