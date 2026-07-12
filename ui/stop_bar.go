package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	stopBarHeight   float32 = 30
	stopValueHeight float32 = 14
	stopHandleW     float32 = 10
	stopHandleH     float32 = 14
	stopGrabPx      float32 = 16 // max x-distance to grab a handle
)

// stopHandle is a draggable tick: a stop id, its normalized position, and the
// stop's color (shown as the handle fill).
type stopHandle struct {
	id  int
	pos float64
	col color.Color
}

// stopBar renders draggable tick handles for gradient stops along a horizontal
// axis. Dragging a handle reports a new normalized position via onDrag; tapping
// a handle selects it via onSelect. The current value is shown above the handle
// only while it is being dragged.
type stopBar struct {
	widget.BaseWidget
	handles  []stopHandle
	selID    int
	dragID   int // id of the handle being dragged, -1 when idle
	dragVal  float64
	onSelect func(id int)
	onDrag   func(id int, pos float64)
	onEnd    func()
}

func newStopBar() *stopBar {
	b := &stopBar{selID: -1, dragID: -1}
	b.ExtendBaseWidget(b)
	return b
}

// SetHandles updates the handles and selection and refreshes.
func (b *stopBar) SetHandles(handles []stopHandle, selID int) {
	b.handles = handles
	b.selID = selID
	b.Refresh()
}

// nearest returns the index of the handle closest to x (within stopGrabPx), or -1.
func (b *stopBar) nearest(x, w float32) int {
	best := -1
	var bestDist float32 = 1e9
	for i, h := range b.handles {
		d := float32(h.pos)*w - x
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	if best >= 0 && bestDist <= stopGrabPx {
		return best
	}
	return -1
}

func (b *stopBar) Tapped(ev *fyne.PointEvent) {
	if i := b.nearest(ev.Position.X, b.Size().Width); i >= 0 && b.onSelect != nil {
		b.onSelect(b.handles[i].id)
	}
}

func (b *stopBar) Dragged(ev *fyne.DragEvent) {
	w := b.Size().Width
	if w <= 0 {
		return
	}
	if b.dragID < 0 {
		i := b.nearest(ev.Position.X, w)
		if i < 0 {
			return
		}
		b.dragID = b.handles[i].id
		if b.onSelect != nil {
			b.onSelect(b.dragID)
		}
	}
	pos := float64(ev.Position.X / w)
	if pos < 0 {
		pos = 0
	} else if pos > 1 {
		pos = 1
	}
	b.dragVal = pos
	if b.onDrag != nil {
		b.onDrag(b.dragID, pos)
	}
}

func (b *stopBar) DragEnd() {
	b.dragID = -1
	if b.onEnd != nil {
		b.onEnd()
	}
	b.Refresh()
}

func (b *stopBar) CreateRenderer() fyne.WidgetRenderer {
	value := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	value.TextSize = theme.CaptionTextSize()
	value.Alignment = fyne.TextAlignCenter
	cont := container.NewWithoutLayout(value)
	r := &stopBarRenderer{bar: b, cont: cont, value: value}
	return r
}

type stopBarRenderer struct {
	bar   *stopBar
	cont  *fyne.Container
	value *canvas.Text
}

func (r *stopBarRenderer) place(size fyne.Size) {
	w := size.Width
	fg := theme.Color(theme.ColorNameForeground)
	sel := theme.Color(theme.ColorNamePrimary)
	objs := make([]fyne.CanvasObject, 0, len(r.bar.handles)+1)
	for _, h := range r.bar.handles {
		rect := canvas.NewRectangle(h.col)
		rect.CornerRadius = 2
		// Outline every handle so light colors stay visible; highlight the
		// selected one with a thicker, accent-colored border.
		rect.StrokeColor = fg
		rect.StrokeWidth = 1
		if h.id == r.bar.selID {
			rect.StrokeColor = sel
			rect.StrokeWidth = 2
		}
		rect.Resize(fyne.NewSize(stopHandleW, stopHandleH))
		x := float32(h.pos)*w - stopHandleW/2
		if x < 0 {
			x = 0
		} else if x > w-stopHandleW {
			x = w - stopHandleW
		}
		rect.Move(fyne.NewPos(x, size.Height-stopHandleH))
		objs = append(objs, rect)
	}
	if r.bar.dragID >= 0 {
		r.value.Text = fmt.Sprintf("%.3f", r.bar.dragVal)
		x := float32(r.bar.dragVal) * w
		r.value.Move(fyne.NewPos(x-24, 0))
		r.value.Resize(fyne.NewSize(48, stopValueHeight))
	} else {
		r.value.Text = ""
	}
	objs = append(objs, r.value)
	r.cont.Objects = objs
}

func (r *stopBarRenderer) Layout(size fyne.Size) {
	r.cont.Resize(size)
	r.place(size)
}

func (r *stopBarRenderer) MinSize() fyne.Size { return fyne.NewSize(120, stopBarHeight) }

func (r *stopBarRenderer) Refresh() {
	r.place(r.bar.Size())
	r.cont.Refresh()
}

func (r *stopBarRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.cont} }

func (r *stopBarRenderer) Destroy() {}

// ensure interfaces
var (
	_ fyne.Draggable = (*stopBar)(nil)
	_ fyne.Tappable  = (*stopBar)(nil)
)
