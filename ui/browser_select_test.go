package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestRowTappedSelects checks the row selects itself on a primary tap. The row
// is SecondaryTappable (for the context menu), so Fyne's tap dispatch targets it
// directly instead of the list's internal wrapper — it must be Tappable too.
func TestRowTappedSelects(t *testing.T) {
	test.NewApp()
	r := newPaletteRow() // compile-time: paletteRow implements fyne.Tappable
	tapped := false
	r.onTapped = func() { tapped = true }
	r.Tapped(&fyne.PointEvent{})
	if !tapped {
		t.Error("Tapped did not fire onTapped")
	}
}

// TestClickSelectsPalette is an end-to-end check: a real positional click on a
// browse row must open that palette in the editor (regression for the bug where
// clicking a row did nothing and the welcome screen stayed).
func TestClickSelectsPalette(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.showWelcome()
	ws.db.SaveDiscrete("clickme", "", niceStops())
	ws.refreshBrowser()

	win := test.NewWindow(ws.browser)
	defer win.Close()
	win.Resize(fyne.NewSize(300, 500))

	// Click down the row column (below the "Custom" header) until the first
	// custom row is hit; it should open the palette.
	c := win.Canvas()
	for y := float32(44); y <= 120 && ws.current == nil; y += 3 {
		test.TapCanvas(c, fyne.NewPos(120, y))
	}
	if ws.current == nil {
		t.Fatal("clicking a row did not open a palette (still on welcome)")
	}
	if ws.current.name != "clickme" {
		t.Errorf("opened %q, want clickme", ws.current.name)
	}
}
