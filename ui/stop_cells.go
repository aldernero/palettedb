package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// swatchCell is a color square in the stop table. A single tap selects the row;
// a double tap opens the color picker.
type swatchCell struct {
	widget.BaseWidget
	rect        *canvas.Rectangle
	onTap       func()
	onDoubleTap func()
}

func newSwatchCell() *swatchCell {
	c := &swatchCell{rect: canvas.NewRectangle(color.Black)}
	c.rect.SetMinSize(fyne.NewSize(22, 18))
	c.rect.CornerRadius = 2
	c.ExtendBaseWidget(c)
	return c
}

func (c *swatchCell) SetColor(col color.Color) {
	c.rect.FillColor = col
	c.rect.Refresh()
}

func (c *swatchCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewCenter(c.rect))
}

func (c *swatchCell) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}

func (c *swatchCell) DoubleTapped(*fyne.PointEvent) {
	if c.onDoubleTap != nil {
		c.onDoubleTap()
	}
}

// posCell shows a stop's position. A single tap selects the row; a double tap
// makes it editable (commit on Enter).
type posCell struct {
	widget.BaseWidget
	label       *widget.Label
	entry       *numericEntry
	editing     bool
	onTap       func()
	onCommit    func(v float64)
	focusCanvas fyne.Canvas
}

func newPosCell() *posCell {
	c := &posCell{label: widget.NewLabel(""), entry: newNumericEntry()}
	c.entry.Hide()
	// numericEntry commits on both Enter and focus loss.
	c.entry.onCommit = func(s string) { c.commit(s) }
	c.ExtendBaseWidget(c)
	return c
}

// commit parses and applies the edited value, then leaves edit mode. It is a
// no-op when not editing, which makes the re-entrant focus-loss triggered by
// hiding the entry in endEdit harmless.
func (c *posCell) commit(s string) {
	if !c.editing {
		return
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil && c.onCommit != nil {
		if v < 0 {
			v = 0
		} else if v > 1 {
			v = 1
		}
		c.onCommit(v)
	}
	c.endEdit()
}

func (c *posCell) SetValue(v float64) {
	c.label.SetText(fmt.Sprintf("%.3f", v))
	if !c.editing {
		c.label.Show()
		c.entry.Hide()
	}
}

func (c *posCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.label, c.entry))
}

func (c *posCell) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}

func (c *posCell) DoubleTapped(*fyne.PointEvent) {
	c.editing = true
	c.entry.SetText(c.label.Text)
	c.label.Hide()
	c.entry.Show()
	c.Refresh()
	if c.focusCanvas != nil {
		c.focusCanvas.Focus(c.entry)
	}
}

func (c *posCell) endEdit() {
	c.editing = false
	c.entry.Hide()
	c.label.Show()
	c.Refresh()
}

var (
	_ fyne.Tappable       = (*swatchCell)(nil)
	_ fyne.DoubleTappable = (*swatchCell)(nil)
	_ fyne.Tappable       = (*posCell)(nil)
	_ fyne.DoubleTappable = (*posCell)(nil)
)
