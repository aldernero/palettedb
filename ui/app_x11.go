//go:build !wayland

package ui

import (
	"time"

	"fyne.io/fyne/v2"
)

// platformWindowInit runs before the window is shown on the X11/XWayland
// backend (the default build).
func platformWindowInit(w fyne.Window) {
	w.CenterOnScreen()
}

// platformAfterShow runs after the window is shown. On XWayland the window is
// laid out at the wrong scale until a configure event arrives, so we nudge a
// relayout to fix the initial HiDPI scaling (gap above the menu bar).
func platformAfterShow(w fyne.Window) {
	nudgeRelayout(w)
}

// nudgeRelayout works around a HiDPI/XWayland issue where the window (and its
// menu bar) is laid out at the wrong scale on first map — leaving a gap above
// the menu — until a configure/rescale event arrives (e.g. dragging the window
// between monitors). Shortly after the window maps we resize it by a pixel and
// back, which forces Fyne to recompute the layout at the monitor's real scale.
// Fyne marshals Resize onto its own thread, so calling it from here is safe.
func nudgeRelayout(w fyne.Window) {
	go func() {
		time.Sleep(250 * time.Millisecond)
		fyne.DoAndWait(func() {
			w.Resize(fyne.NewSize(initialWindowSize.Width, initialWindowSize.Height+1))
		})
		time.Sleep(50 * time.Millisecond)
		fyne.DoAndWait(func() {
			w.Resize(initialWindowSize)
		})
	}()
}
