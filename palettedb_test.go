package palettedb

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

// seed writes a custom sine and discrete palette, then closes the DB so the
// public API can reopen the same file.
func seed(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "p.db")
	inner, err := db.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inner.SaveSine("my-sine", "", gaul.NewSinePalette(
		gaul.Vec3{X: 1, Y: 0.7, Z: 0.4}, gaul.Vec3{X: 0, Y: 0.15, Z: 0.2})); err != nil {
		t.Fatal(err)
	}
	if _, err := inner.SaveDiscrete("my-grad", "", []db.ColorStop{
		{Color: color.RGBA{255, 0, 0, 255}, Pos: 0},
		{Color: color.RGBA{0, 0, 255, 255}, Pos: 1},
	}); err != nil {
		t.Fatal(err)
	}
	inner.Close()
	return path
}

func TestByNameSearchesCustomAndBuiltin(t *testing.T) {
	d, err := Open(seed(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// Custom sine + discrete resolve.
	if _, err := d.LoadSineByName("my-sine"); err != nil {
		t.Errorf("custom sine not found: %v", err)
	}
	if _, err := d.LoadDiscreteByName("my-grad"); err != nil {
		t.Errorf("custom discrete not found: %v", err)
	}

	// Built-in discrete resolves by name (case-insensitively) even though it is
	// not in the database.
	g, err := d.LoadDiscreteByName("Viridis")
	if err != nil {
		t.Fatalf("built-in viridis not found: %v", err)
	}
	r, gr, b, _ := g.ColorAt(0).RGBA()
	if to8(r) != 0x44 || to8(gr) != 0x01 || to8(b) != 0x54 {
		t.Errorf("viridis start = #%02X%02X%02X, want #440154", to8(r), to8(gr), to8(b))
	}

	// Built-in sine palette resolves (not in the database).
	if _, err := d.LoadSineByName("warm-sunset"); err != nil {
		t.Errorf("built-in sine warm-sunset not found: %v", err)
	}

	// PaletteByName reports the type.
	if _, kind, err := d.PaletteByName("turbo"); err != nil || kind != "discrete" {
		t.Errorf("PaletteByName(turbo) = %q, %v; want discrete", kind, err)
	}
	if _, kind, err := d.PaletteByName("warm-sunset"); err != nil || kind != "sine" {
		t.Errorf("PaletteByName(warm-sunset) = %q, %v; want sine", kind, err)
	}

	// Wrong type / missing name errors.
	if _, err := d.LoadSineByName("viridis"); err == nil {
		t.Error("expected error: viridis is discrete, not sine")
	}
	if _, err := d.LoadDiscreteByName("does-not-exist"); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestListStoredPalettes(t *testing.T) {
	d, err := Open(seed(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	entries, err := d.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2", len(entries))
	}
	// ListAll orders by name: my-grad before my-sine.
	if entries[0].Name != "my-grad" || entries[0].Type != "discrete" {
		t.Errorf("entries[0] = %+v, want my-grad/discrete", entries[0])
	}
	if entries[1].Name != "my-sine" || entries[1].Type != "sine" {
		t.Errorf("entries[1] = %+v, want my-sine/sine", entries[1])
	}

	sines, err := d.ListNames("sine")
	if err != nil {
		t.Fatal(err)
	}
	if len(sines) != 1 || sines[0] != "my-sine" {
		t.Errorf("ListNames(sine) = %v, want [my-sine]", sines)
	}
	discretes, err := d.ListNames("discrete")
	if err != nil {
		t.Fatal(err)
	}
	if len(discretes) != 1 || discretes[0] != "my-grad" {
		t.Errorf("ListNames(discrete) = %v, want [my-grad]", discretes)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "palettedb", "palettedb.db")
	if p != want {
		t.Errorf("DefaultPath() = %q, want %q", p, want)
	}
}

func to8(x uint32) uint8 { return uint8((x*255 + 32767) / 65535) }
