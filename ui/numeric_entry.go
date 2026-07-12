package ui

import "fyne.io/fyne/v2/widget"

// numericEntry is a single-line entry that commits its text both on Enter and on
// focus loss. Fyne's widget.Entry only fires OnSubmitted on Enter, so a value
// typed and then clicked/tabbed away from would otherwise be dropped.
type numericEntry struct {
	widget.Entry
	onCommit func(text string)
}

func newNumericEntry() *numericEntry {
	e := &numericEntry{}
	e.ExtendBaseWidget(e)
	e.OnSubmitted = func(s string) {
		if e.onCommit != nil {
			e.onCommit(s)
		}
	}
	return e
}

// FocusLost commits the current text (in addition to the base behavior).
func (e *numericEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onCommit != nil {
		e.onCommit(e.Text)
	}
}
