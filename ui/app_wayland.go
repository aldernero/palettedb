//go:build wayland

package ui

import (
	"bytes"
	"io"
	"log"

	"fyne.io/fyne/v2"
)

// platformWindowInit runs before the window is shown on the native Wayland
// backend (built with -tags wayland). Wayland handles per-monitor scaling
// correctly, so no centering or relayout nudge is needed. We do filter one
// benign GLFW log line (see below).
func platformWindowInit(_ fyne.Window) {
	suppressBenignWaylandLogs()
}

// platformAfterShow is a no-op on Wayland.
func platformAfterShow(_ fyne.Window) {}

// suppressBenignWaylandLogs drops the harmless GLFW message emitted when Fyne
// shows a window on Wayland ("Focusing a window requires user interaction").
// Wayland forbids clients from self-focusing without an activation token, so
// GLFW's focus-on-show always logs this once at startup. Everything else passes
// through unchanged.
func suppressBenignWaylandLogs() {
	log.SetOutput(focusLogFilter{out: log.Writer()})
}

type focusLogFilter struct{ out io.Writer }

func (f focusLogFilter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("Focusing a window requires user interaction")) {
		return len(p), nil
	}
	return f.out.Write(p)
}
