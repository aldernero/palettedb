package db

import (
	"image/color"
	"math"
	"path/filepath"
	"testing"

	"github.com/aldernero/gaul"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := OpenAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestSaveLoadSine(t *testing.T) {
	d := openTestDB(t)
	sp := gaul.SinePalette{
		A:     gaul.Vec3{X: 0.5, Y: 0.5, Z: 0.5},
		B:     gaul.Vec3{X: 0.5, Y: 0.5, Z: 0.5},
		C:     gaul.Vec3{X: 1.0, Y: 0.7, Z: 0.3},
		D:     gaul.Vec3{X: 0.0, Y: 0.15, Z: 0.2},
		Alpha: 1.0,
	}
	id, err := d.SaveSine("test-sine", "a test", sp)
	if err != nil {
		t.Fatalf("SaveSine: %v", err)
	}
	got, err := d.LoadSine(id)
	if err != nil {
		t.Fatalf("LoadSine: %v", err)
	}
	eps := 1e-10
	for _, pair := range [][2]float64{
		{sp.A.X, got.Palette.A.X}, {sp.A.Y, got.Palette.A.Y}, {sp.A.Z, got.Palette.A.Z},
		{sp.B.X, got.Palette.B.X}, {sp.B.Y, got.Palette.B.Y}, {sp.B.Z, got.Palette.B.Z},
		{sp.C.X, got.Palette.C.X}, {sp.C.Y, got.Palette.C.Y}, {sp.C.Z, got.Palette.C.Z},
		{sp.D.X, got.Palette.D.X}, {sp.D.Y, got.Palette.D.Y}, {sp.D.Z, got.Palette.D.Z},
		{sp.Alpha, got.Palette.Alpha},
	} {
		if math.Abs(pair[0]-pair[1]) > eps {
			t.Errorf("float mismatch: want %v got %v", pair[0], pair[1])
		}
	}
	if got.Name != "test-sine" {
		t.Errorf("name mismatch: %q", got.Name)
	}
}

func TestSaveLoadDiscrete(t *testing.T) {
	d := openTestDB(t)
	stops := []ColorStop{
		{Color: color.RGBA64{R: 65535, G: 0, B: 0, A: 65535}, Pos: 0}, // pure red
		{Color: color.RGBA64{R: 0, G: 0, B: 65535, A: 65535}, Pos: 1}, // pure blue
	}
	id, err := d.SaveDiscrete("test-discrete", "red to blue", stops)
	if err != nil {
		t.Fatalf("SaveDiscrete: %v", err)
	}
	got, err := d.LoadDiscrete(id)
	if err != nil {
		t.Fatalf("LoadDiscrete: %v", err)
	}
	if len(got.Colors()) != 2 {
		t.Fatalf("expected 2 colors, got %d", len(got.Colors()))
	}
	g := got.Gradient()
	r0, g0, b0, _ := g.ColorAt(0).RGBA()
	if r0 < 60000 || g0 > 5000 || b0 > 5000 {
		t.Errorf("ColorAt(0) should be red, got R=%d G=%d B=%d", r0, g0, b0)
	}
	r1, g1, b1, _ := g.ColorAt(1).RGBA()
	if r1 > 5000 || g1 > 5000 || b1 < 60000 {
		t.Errorf("ColorAt(1) should be blue, got R=%d G=%d B=%d", r1, g1, b1)
	}
}

func TestSaveLoadDiscretePositions(t *testing.T) {
	d := openTestDB(t)
	stops := []ColorStop{
		{Color: color.RGBA64{R: 65535, A: 65535}, Pos: 0},
		{Color: color.RGBA64{G: 65535, A: 65535}, Pos: 0.25},
		{Color: color.RGBA64{B: 65535, A: 65535}, Pos: 1},
	}
	id, err := d.SaveDiscrete("test-pos", "uneven", stops)
	if err != nil {
		t.Fatalf("SaveDiscrete: %v", err)
	}
	got, err := d.LoadDiscrete(id)
	if err != nil {
		t.Fatalf("LoadDiscrete: %v", err)
	}
	want := []float64{0, 0.25, 1}
	ps := got.Positions()
	if len(ps) != len(want) {
		t.Fatalf("expected %d positions, got %d", len(want), len(ps))
	}
	for i := range want {
		if math.Abs(ps[i]-want[i]) > 1e-9 {
			t.Errorf("position %d: want %v got %v", i, want[i], ps[i])
		}
	}
}

// TestLegacyPositionBackfill simulates a pre-migration row (NULL pos) and checks
// that LoadDiscrete backfills evenly spaced positions.
func TestLegacyPositionBackfill(t *testing.T) {
	d := openTestDB(t)
	res, err := d.Exec(`INSERT INTO directory(name, type, description) VALUES ('legacy','discrete','')`)
	if err != nil {
		t.Fatalf("insert directory: %v", err)
	}
	id, _ := res.LastInsertId()
	if _, err := d.Exec(`INSERT INTO discrete_palettes(id) VALUES (?)`, id); err != nil {
		t.Fatalf("insert palette: %v", err)
	}
	// Insert three stops with NULL pos (pos column omitted).
	for i := 0; i < 3; i++ {
		if _, err := d.Exec(`INSERT INTO discrete_colors(palette_id, position, r, g, b) VALUES (?,?,?,?,?)`,
			id, i, 0.1*float64(i), 0.2, 0.3); err != nil {
			t.Fatalf("insert color: %v", err)
		}
	}
	got, err := d.LoadDiscrete(id)
	if err != nil {
		t.Fatalf("LoadDiscrete: %v", err)
	}
	want := []float64{0, 0.5, 1}
	ps := got.Positions()
	for i := range want {
		if math.Abs(ps[i]-want[i]) > 1e-9 {
			t.Errorf("backfilled position %d: want %v got %v", i, want[i], ps[i])
		}
	}
}

func TestDelete(t *testing.T) {
	d := openTestDB(t)
	sp := gaul.NewSinePalette(
		gaul.Vec3{X: 1, Y: 0.7, Z: 0.3},
		gaul.Vec3{X: 0, Y: 0.15, Z: 0.2},
	)
	id, err := d.SaveSine("to-delete", "", sp)
	if err != nil {
		t.Fatalf("SaveSine: %v", err)
	}
	if err := d.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	entries, err := d.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
	var count int
	d.QueryRow(`SELECT count(*) FROM sine_palettes WHERE id=?`, id).Scan(&count)
	if count != 0 {
		t.Errorf("sine_palettes row not cascade-deleted")
	}
}
