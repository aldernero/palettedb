package ui

import (
	"fmt"
	"image/color"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

// stopItem is a color stop with a stable id so the selection can survive
// re-sorting when positions change.
type stopItem struct {
	id    int
	color color.Color
	pos   float64
}

// DiscreteEditor edits a discrete palette: a set of color stops at adjustable
// positions. Name/description/saving are handled by the Workspace controller.
type DiscreteEditor struct {
	widget.BaseWidget
	window fyne.Window

	stops    []stopItem
	nextID   int
	selID    int // id of the selected stop, -1 if none
	OnChange func()

	view *PaletteView
	bar  *stopBar
	list *widget.List
}

func NewDiscreteEditor(window fyne.Window) *DiscreteEditor {
	e := &DiscreteEditor{window: window, selID: -1}
	e.view = NewPaletteView(nil)
	e.reset()
	e.ExtendBaseWidget(e)
	return e
}

// reset restores the default two-stop black→white palette.
func (e *DiscreteEditor) reset() {
	e.stops = nil
	e.nextID = 0
	e.addStopValue(color.Black, 0)
	e.addStopValue(color.White, 1)
	e.selID = e.stops[0].id
	e.refreshAll()
}

func (e *DiscreteEditor) addStopValue(c color.Color, pos float64) {
	e.stops = append(e.stops, stopItem{id: e.nextID, color: c, pos: pos})
	e.nextID++
}

func (e *DiscreteEditor) sortStops() {
	sort.SliceStable(e.stops, func(i, j int) bool { return e.stops[i].pos < e.stops[j].pos })
}

func (e *DiscreteEditor) selectedIndex() int {
	for i := range e.stops {
		if e.stops[i].id == e.selID {
			return i
		}
	}
	return -1
}

// Stops returns the current stops as db.ColorStop, sorted by position.
func (e *DiscreteEditor) Stops() []db.ColorStop {
	e.sortStops()
	out := make([]db.ColorStop, len(e.stops))
	for i, s := range e.stops {
		out[i] = db.ColorStop{Color: s.color, Pos: s.pos}
	}
	return out
}

// LoadStops populates the editor from an existing palette's stops.
func (e *DiscreteEditor) LoadStops(stops []db.ColorStop) {
	e.stops = nil
	e.nextID = 0
	for _, s := range stops {
		e.addStopValue(s.Color, s.Pos)
	}
	if len(e.stops) == 0 {
		e.reset()
		return
	}
	e.sortStops()
	e.selID = e.stops[0].id
	e.refreshAll()
}

// Gradient returns the current position-aware gradient (for preview/export).
func (e *DiscreteEditor) Gradient() gaul.Gradient { return e.buildGradient() }

func (e *DiscreteEditor) buildGradient() gaul.Gradient {
	e.sortStops()
	colors := make([]color.Color, len(e.stops))
	positions := make([]float64, len(e.stops))
	for i, s := range e.stops {
		colors[i] = s.color
		positions[i] = s.pos
	}
	return gaul.NewGradientFromColorStops(colors, positions)
}

// handleList returns the current stops as draggable handles (sorted order).
func (e *DiscreteEditor) handleList() []stopHandle {
	hs := make([]stopHandle, len(e.stops))
	for i, s := range e.stops {
		hs[i] = stopHandle{id: s.id, pos: s.pos, col: s.color}
	}
	return hs
}

// refreshAll rebuilds the preview, tick bar, and stop table.
func (e *DiscreteEditor) refreshAll() {
	g := e.buildGradient() // also sorts e.stops
	e.view.SetPalette(&g)
	if e.bar != nil {
		e.bar.SetHandles(e.handleList(), e.selID)
	}
	if e.list != nil {
		e.list.Refresh()
		if idx := e.selectedIndex(); idx >= 0 {
			e.list.Select(idx)
		}
	}
	if e.OnChange != nil {
		e.OnChange()
	}
}

// selectByID selects the stop with the given id (highlighting the table row and
// the tick). List.OnSelected applies the side effects.
func (e *DiscreteEditor) selectByID(id int) {
	if e.list == nil {
		e.selID = id
		return
	}
	for i := range e.stops {
		if e.stops[i].id == id {
			e.list.Select(i)
			return
		}
	}
}

// setStopPosByID sets a stop's position (used by the tick bar and inline edit).
func (e *DiscreteEditor) setStopPosByID(id int, pos float64) {
	for i := range e.stops {
		if e.stops[i].id == id {
			e.stops[i].pos = pos
			break
		}
	}
	e.refreshAll()
}

// editStopColor opens the color picker for the stop with the given id.
func (e *DiscreteEditor) editStopColor(id int) {
	var cur color.Color = color.Black
	for i := range e.stops {
		if e.stops[i].id == id {
			cur = e.stops[i].color
			break
		}
	}
	ShowColorPickerDialog("Pick color", cur, func(c color.Color) {
		for i := range e.stops {
			if e.stops[i].id == id {
				e.stops[i].color = c
				break
			}
		}
		e.refreshAll()
	}, e.window)
}

func (e *DiscreteEditor) addStop() {
	idx := e.selectedIndex()
	var pos float64
	var c color.Color
	if idx < 0 {
		pos, c = 0.5, color.White
	} else if idx+1 < len(e.stops) {
		// Insert halfway to the next stop.
		pos = (e.stops[idx].pos + e.stops[idx+1].pos) / 2
		c = e.stops[idx].color
	} else if idx-1 >= 0 {
		pos = (e.stops[idx].pos + e.stops[idx-1].pos) / 2
		c = e.stops[idx].color
	} else {
		pos, c = 0.5, e.stops[idx].color
	}
	e.addStopValue(c, pos)
	e.selID = e.stops[len(e.stops)-1].id
	e.refreshAll()
}

func (e *DiscreteEditor) removeStop() {
	if len(e.stops) <= 2 {
		return
	}
	idx := e.selectedIndex()
	if idx < 0 {
		return
	}
	e.stops = append(e.stops[:idx], e.stops[idx+1:]...)
	// Select a neighbor.
	if idx >= len(e.stops) {
		idx = len(e.stops) - 1
	}
	e.selID = e.stops[idx].id
	e.refreshAll()
}

// hexString formats a color as #RRGGBB.
func hexString(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", to8(r), to8(g), to8(b))
}

// to8 converts a 16-bit color channel (0..65535) to 8 bits, rounding to nearest
// (matching the round(v*255) convention used by matplotlib and most tools).
func to8(x uint32) uint8 { return uint8((x*255 + 32767) / 65535) }

// stop-table column widths (must match between header and rows).
var stopCols = struct{ idx, color, hex, pos float32 }{34, 48, 96, 74}

// fixedWidthLayout pins a column to a fixed width while keeping its natural
// height (so text is not vertically clipped).
type fixedWidthLayout struct{ w float32 }

func (l fixedWidthLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objs {
		if mh := o.MinSize().Height; mh > h {
			h = mh
		}
	}
	return fyne.NewSize(l.w, h)
}

func (l fixedWidthLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(fyne.NewSize(l.w, size.Height))
		o.Move(fyne.NewPos(0, 0))
	}
}

func fixedW(w float32, o fyne.CanvasObject) fyne.CanvasObject {
	return container.New(fixedWidthLayout{w}, o)
}

func (e *DiscreteEditor) CreateRenderer() fyne.WidgetRenderer {
	e.bar = newStopBar()
	e.bar.onSelect = e.selectByID
	e.bar.onDrag = e.setStopPosByID
	e.bar.onEnd = func() { e.refreshAll() }

	e.list = widget.NewList(
		func() int { return len(e.stops) },
		func() fyne.CanvasObject {
			idx := widget.NewLabel("")
			sw := newSwatchCell()
			hex := widget.NewLabel("")
			pos := newPosCell()
			pos.focusCanvas = e.window.Canvas()
			return container.NewHBox(
				fixedW(stopCols.idx, idx),
				fixedW(stopCols.color, sw),
				fixedW(stopCols.hex, hex),
				fixedW(stopCols.pos, pos),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			idxLbl := row.Objects[0].(*fyne.Container).Objects[0].(*widget.Label)
			sw := row.Objects[1].(*fyne.Container).Objects[0].(*swatchCell)
			hexLbl := row.Objects[2].(*fyne.Container).Objects[0].(*widget.Label)
			pos := row.Objects[3].(*fyne.Container).Objects[0].(*posCell)
			s := e.stops[id]
			sid := s.id
			idxLbl.SetText(fmt.Sprintf("%d", id+1))
			hexLbl.SetText(hexString(s.color))
			sw.SetColor(s.color)
			sw.onTap = func() { e.selectByID(sid) }
			sw.onDoubleTap = func() { e.editStopColor(sid) }
			pos.SetValue(s.pos)
			pos.onTap = func() { e.selectByID(sid) }
			pos.onCommit = func(v float64) { e.setStopPosByID(sid, v) }
		},
	)
	e.list.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(e.stops) {
			e.selID = e.stops[id].id
			if e.bar != nil {
				e.bar.SetHandles(e.handleList(), e.selID)
			}
		}
	}

	bold := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	header := container.NewHBox(
		fixedW(stopCols.idx, bold("Stop")),
		fixedW(stopCols.color, bold("Color")),
		fixedW(stopCols.hex, bold("Hex")),
		fixedW(stopCols.pos, bold("Position")),
	)
	hint := widget.NewLabel("Double-click a color to change it, or a position to edit it.")

	addBtn := widget.NewButtonWithIcon("Add", nil, e.addStop)
	removeBtn := widget.NewButtonWithIcon("Remove", nil, e.removeStop)
	addRemove := container.NewGridWithColumns(2, addBtn, removeBtn)

	listScroll := container.NewVScroll(e.list)
	listScroll.SetMinSize(fyne.NewSize(0, 180))

	content := container.NewVBox(
		e.bar,
		e.view,
		widget.NewSeparator(),
		widget.NewLabel("Color stops:"),
		header,
		listScroll,
		addRemove,
		hint,
	)
	e.refreshAll()
	return widget.NewSimpleRenderer(content)
}
