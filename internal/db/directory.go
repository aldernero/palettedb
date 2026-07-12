package db

// Entry is a row from the directory table. BuiltIn is not persisted; it is set
// by the UI for synthetic entries representing read-only built-in palettes.
type Entry struct {
	ID          int64
	Name        string
	Type        string // "sine" or "discrete"
	Description string
	BuiltIn     bool
}

// ListAll returns all directory entries ordered by name.
func (d *DB) ListAll() ([]Entry, error) {
	rows, err := d.Query(`SELECT id, name, type, description FROM directory ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Description); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Get returns a single directory entry by ID.
func (d *DB) Get(id int64) (Entry, error) {
	var e Entry
	err := d.QueryRow(`SELECT id, name, type, description FROM directory WHERE id = ?`, id).
		Scan(&e.ID, &e.Name, &e.Type, &e.Description)
	return e, err
}

// GetByName returns a single directory entry by name.
func (d *DB) GetByName(name string) (Entry, error) {
	var e Entry
	err := d.QueryRow(`SELECT id, name, type, description FROM directory WHERE name = ?`, name).
		Scan(&e.ID, &e.Name, &e.Type, &e.Description)
	return e, err
}

// Rename updates the name and description of a directory entry.
func (d *DB) Rename(id int64, name, description string) error {
	_, err := d.Exec(`UPDATE directory SET name = ?, description = ? WHERE id = ?`,
		name, description, id)
	return err
}

// Delete removes a directory entry and cascades to the associated subtype table.
func (d *DB) Delete(id int64) error {
	_, err := d.Exec(`DELETE FROM directory WHERE id = ?`, id)
	return err
}
