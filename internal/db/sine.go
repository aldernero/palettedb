package db

import "github.com/aldernero/gaul"

func colorSpaceName(cs gaul.ColorSpace) string {
	if cs == gaul.ColorSpaceHSV {
		return "HSV"
	}
	return "RGB"
}

func parseColorSpace(s string) gaul.ColorSpace {
	if s == "HSV" {
		return gaul.ColorSpaceHSV
	}
	return gaul.ColorSpaceRGB
}

// SinePaletteRecord wraps a gaul.SinePalette with its directory metadata.
type SinePaletteRecord struct {
	Entry
	Palette gaul.SinePalette
}

// SaveSine inserts a directory entry and a sine_palettes row in one transaction.
func (d *DB) SaveSine(name, description string, sp gaul.SinePalette) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO directory(name, type, description) VALUES (?,?,?)`,
		name, "sine", description)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(`INSERT INTO sine_palettes(id,ax,ay,az,bx,by,bz,cx,cy,cz,dx,dy,dz,alpha,color_space)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		sp.A.X, sp.A.Y, sp.A.Z,
		sp.B.X, sp.B.Y, sp.B.Z,
		sp.C.X, sp.C.Y, sp.C.Z,
		sp.D.X, sp.D.Y, sp.D.Z,
		sp.Alpha,
		colorSpaceName(sp.Space),
	)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// LoadSine loads a SinePaletteRecord by directory ID.
func (d *DB) LoadSine(id int64) (SinePaletteRecord, error) {
	entry, err := d.Get(id)
	if err != nil {
		return SinePaletteRecord{}, err
	}
	var sp gaul.SinePalette
	var spaceName string
	err = d.QueryRow(`SELECT ax,ay,az,bx,by,bz,cx,cy,cz,dx,dy,dz,alpha,color_space FROM sine_palettes WHERE id = ?`, id).
		Scan(
			&sp.A.X, &sp.A.Y, &sp.A.Z,
			&sp.B.X, &sp.B.Y, &sp.B.Z,
			&sp.C.X, &sp.C.Y, &sp.C.Z,
			&sp.D.X, &sp.D.Y, &sp.D.Z,
			&sp.Alpha,
			&spaceName,
		)
	if err != nil {
		return SinePaletteRecord{}, err
	}
	sp.Space = parseColorSpace(spaceName)
	return SinePaletteRecord{Entry: entry, Palette: sp}, nil
}

// UpdateSine replaces the palette parameters for an existing sine palette.
func (d *DB) UpdateSine(id int64, sp gaul.SinePalette) error {
	_, err := d.Exec(`UPDATE sine_palettes
		SET ax=?,ay=?,az=?,bx=?,by=?,bz=?,cx=?,cy=?,cz=?,dx=?,dy=?,dz=?,alpha=?,color_space=?
		WHERE id=?`,
		sp.A.X, sp.A.Y, sp.A.Z,
		sp.B.X, sp.B.Y, sp.B.Z,
		sp.C.X, sp.C.Y, sp.C.Z,
		sp.D.X, sp.D.Y, sp.D.Z,
		sp.Alpha,
		colorSpaceName(sp.Space),
		id,
	)
	return err
}

// ListSine returns all directory entries of type "sine" with their palettes loaded.
func (d *DB) ListSine() ([]SinePaletteRecord, error) {
	entries, err := d.ListAll()
	if err != nil {
		return nil, err
	}
	var records []SinePaletteRecord
	for _, e := range entries {
		if e.Type != "sine" {
			continue
		}
		r, err := d.LoadSine(e.ID)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
