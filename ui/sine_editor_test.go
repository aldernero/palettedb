package ui

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestSineEntryCommits verifies a typed axis value updates the slider and model
// both when submitted with Enter and when the entry loses focus (clicked away).
func TestSineEntryCommits(t *testing.T) {
	test.NewApp()
	e := NewSineEditor(test.NewWindow(nil))
	w := test.NewWindow(e)
	defer w.Close()
	w.Resize(fyne.NewSize(700, 700))
	g := &e.groups[0] // group A, range [-7,7]

	// Enter / OnSubmitted.
	g.entries[0].OnSubmitted("2.5")
	if math.Abs(float64(g.sliders[0].Value)-2.5) > 1e-9 {
		t.Errorf("after Enter: X slider = %v, want 2.5", g.sliders[0].Value)
	}
	if math.Abs(e.palette.A.X-2.5) > 1e-9 {
		t.Errorf("after Enter: A.X = %v, want 2.5", e.palette.A.X)
	}

	// Focus loss (type then click away, no Enter).
	g.entries[1].SetText("3.25")
	g.entries[1].FocusLost()
	if math.Abs(float64(g.sliders[1].Value)-3.25) > 1e-9 {
		t.Errorf("after blur: Y slider = %v, want 3.25", g.sliders[1].Value)
	}
	if math.Abs(e.palette.A.Y-3.25) > 1e-9 {
		t.Errorf("after blur: A.Y = %v, want 3.25", e.palette.A.Y)
	}
}
