package ui

import (
	"image/color"
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

func snap(t *testing.T, ws *Workspace, path string) {
	t.Helper()
	ws.win.Resize(fyne.NewSize(1000, 760))
	img := ws.win.Canvas().Capture()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func niceStops() []db.ColorStop {
	return []db.ColorStop{
		{Color: color.RGBA{0x2E, 0x1A, 0x47, 0xFF}, Pos: 0.0},
		{Color: color.RGBA{0xC0, 0x39, 0x2B, 0xFF}, Pos: 0.38},
		{Color: color.RGBA{0xE6, 0x7E, 0x22, 0xFF}, Pos: 0.7},
		{Color: color.RGBA{0xF1, 0xC4, 0x0F, 0xFF}, Pos: 1.0},
	}
}

func TestGenScreenshots(t *testing.T) {
	if os.Getenv("GEN_SCREENSHOTS") == "" {
		t.Skip("set GEN_SCREENSHOTS=1 to regenerate docs screenshots")
	}
	if err := os.MkdirAll("../docs", 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Browse: a couple of custom palettes + a built-in read-only preview.
	{
		ws := newTestWorkspace(t)
		ws.db.SaveDiscrete("sunset", "", niceStops())
		ws.db.SaveSine("ocean", "", gaul.NewSinePalette(
			gaul.Vec3{X: 0.5, Y: 0.6, Z: 0.9}, gaul.Vec3{X: 0.3, Y: 0.2, Z: 0.2}))
		ws.refreshBrowser()
		for _, it := range ws.browser.builtinItems {
			if it.name == "viridis" {
				ws.onBrowseSelect(it)
			}
		}
		snap(t, ws, "../docs/browse.png")
	}

	// 2. Sine editor: an editable copy of the rocketpop built-in.
	{
		ws := newTestWorkspace(t)
		for _, it := range ws.browser.builtinItems {
			if it.name == "rocketpop" {
				ws.makeCopy(it)
			}
		}
		snap(t, ws, "../docs/sine.png")
	}

	// 3. Discrete editor: a small hand-built gradient.
	{
		ws := newTestWorkspace(t)
		id := ws.newSeq
		ws.newSeq--
		d := &document{id: id, kind: docDiscrete, name: "my-gradient", dirty: true, stops: niceStops()}
		ws.docs[id] = d
		ws.loadDoc(d)
		ws.refreshBrowser()
		ws.browser.SelectByID(id)
		snap(t, ws, "../docs/discrete.png")
	}

	// 4. Welcome screen.
	{
		ws := newTestWorkspace(t)
		ws.showWelcome()
		snap(t, ws, "../docs/welcome.png")
	}
}
