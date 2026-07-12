package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestSwatchCellTaps(t *testing.T) {
	test.NewApp()
	c := newSwatchCell()
	c.SetColor(color.RGBA{R: 255, A: 255})
	tapped, doubled := false, false
	c.onTap = func() { tapped = true }
	c.onDoubleTap = func() { doubled = true }

	test.Tap(c)
	if !tapped {
		t.Error("single tap did not fire onTap")
	}
	test.DoubleTap(c)
	if !doubled {
		t.Error("double tap did not fire onDoubleTap (color picker)")
	}
}

func TestPosCellDoubleTapEdits(t *testing.T) {
	test.NewApp()
	c := newPosCell()
	w := test.NewWindow(c)
	defer w.Close()
	c.SetValue(0.25)

	var committed float64 = -1
	c.onCommit = func(v float64) { committed = v }

	// Not editable until double-tapped.
	if c.editing || c.entry.Visible() {
		t.Fatal("expected label mode before double tap")
	}
	test.DoubleTap(c)
	if !c.editing || !c.entry.Visible() {
		t.Fatal("expected editable entry after double tap")
	}

	// Commit a new value via the entry's submit handler.
	c.entry.SetText("0.42")
	c.entry.OnSubmitted(c.entry.Text)
	if committed < 0.419 || committed > 0.421 {
		t.Errorf("commit = %v, want 0.42", committed)
	}
	if c.editing || c.entry.Visible() {
		t.Error("expected to leave edit mode after commit")
	}
}

func TestPosCellClampsOnCommit(t *testing.T) {
	test.NewApp()
	c := newPosCell()
	w := test.NewWindow(c)
	defer w.Close()
	var committed float64 = -1
	c.onCommit = func(v float64) { committed = v }
	test.DoubleTap(c)
	c.entry.SetText("5")
	c.entry.OnSubmitted(c.entry.Text)
	if committed != 1 {
		t.Errorf("out-of-range value should clamp to 1, got %v", committed)
	}
}

// TestPosCellCommitsOnBlur checks that a value typed and then clicked/tabbed away
// from (focus lost, no Enter) is still committed.
func TestPosCellCommitsOnBlur(t *testing.T) {
	test.NewApp()
	c := newPosCell()
	w := test.NewWindow(c)
	defer w.Close()
	var committed float64 = -1
	c.onCommit = func(v float64) { committed = v }

	test.DoubleTap(c) // enter edit mode
	c.entry.SetText("0.6")
	c.entry.FocusLost() // blur without pressing Enter

	if committed < 0.599 || committed > 0.601 {
		t.Errorf("blur commit = %v, want 0.6", committed)
	}
	if c.editing {
		t.Error("expected to leave edit mode after blur commit")
	}
}
