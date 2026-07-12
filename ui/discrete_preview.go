package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

// DiscretePreview is a read-only view of a discrete palette: a gradient/swatch
// preview plus the full stops table (no ticks, no editing). Used to display
// built-in colormaps.
type DiscretePreview struct {
	widget.BaseWidget
	stops []db.ColorStop
	view  *PaletteView
	list  *widget.List
}

func NewDiscretePreview() *DiscretePreview {
	p := &DiscretePreview{view: NewPaletteView(nil)}
	p.ExtendBaseWidget(p)
	return p
}

// SetStops populates the preview from a palette's stops.
func (p *DiscretePreview) SetStops(stops []db.ColorStop) {
	p.stops = stops
	colors := make([]color.Color, len(stops))
	positions := make([]float64, len(stops))
	for i, s := range stops {
		colors[i] = s.Color
		positions[i] = s.Pos
	}
	g := gaul.NewGradientFromColorStops(colors, positions)
	p.view.SetPalette(&g)
	if p.list != nil {
		p.list.Refresh()
	}
}

func (p *DiscretePreview) CreateRenderer() fyne.WidgetRenderer {
	p.list = widget.NewList(
		func() int { return len(p.stops) },
		func() fyne.CanvasObject {
			sw := canvas.NewRectangle(color.Black)
			sw.SetMinSize(fyne.NewSize(22, 18))
			return container.NewHBox(
				fixedW(stopCols.idx, widget.NewLabel("")),
				fixedW(stopCols.color, container.NewCenter(sw)),
				fixedW(stopCols.hex, widget.NewLabel("")),
				fixedW(stopCols.pos, widget.NewLabel("")),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			idx := row.Objects[0].(*fyne.Container).Objects[0].(*widget.Label)
			sw := row.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*canvas.Rectangle)
			hex := row.Objects[2].(*fyne.Container).Objects[0].(*widget.Label)
			pos := row.Objects[3].(*fyne.Container).Objects[0].(*widget.Label)
			s := p.stops[id]
			idx.SetText(fmt.Sprintf("%d", id+1))
			sw.FillColor = s.Color
			sw.Refresh()
			hex.SetText(hexString(s.Color))
			pos.SetText(fmt.Sprintf("%.3f", s.Pos))
		},
	)

	bold := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	header := container.NewHBox(
		fixedW(stopCols.idx, bold("Stop")),
		fixedW(stopCols.color, bold("Color")),
		fixedW(stopCols.hex, bold("Hex")),
		fixedW(stopCols.pos, bold("Position")),
	)

	// The gradient preview + header sit on top; the stops table fills the rest of
	// the available height (it scrolls internally).
	top := container.NewVBox(
		p.view,
		widget.NewSeparator(),
		widget.NewLabel("Color stops (read-only):"),
		header,
	)
	content := container.NewBorder(top, nil, nil, nil, p.list)
	return widget.NewSimpleRenderer(content)
}
