package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
)

// SinePreview is a read-only view of a sine palette: a gradient/swatch preview
// plus the A/B/C/D coefficients (no editing). Used to display built-in sine
// palettes.
type SinePreview struct {
	widget.BaseWidget
	palette   gaul.SinePalette
	view      *PaletteView
	groupLbls [4]*widget.Label
	alphaLbl  *widget.Label
	spaceLbl  *widget.Label
}

func NewSinePreview() *SinePreview {
	p := &SinePreview{view: NewPaletteView(nil)}
	for i := range p.groupLbls {
		p.groupLbls[i] = widget.NewLabel("")
	}
	p.alphaLbl = widget.NewLabel("")
	p.spaceLbl = widget.NewLabel("")
	p.ExtendBaseWidget(p)
	return p
}

// SetPalette populates the preview from a sine palette.
func (p *SinePreview) SetPalette(sp gaul.SinePalette) {
	p.palette = sp
	p.view.SetPalette(&p.palette)
	vecs := [4]gaul.Vec3{sp.A, sp.B, sp.C, sp.D}
	for i, v := range vecs {
		p.groupLbls[i].SetText(fmt.Sprintf("X %.3f      Y %.3f      Z %.3f", v.X, v.Y, v.Z))
	}
	p.alphaLbl.SetText(fmt.Sprintf("%.3f", sp.Alpha))
	space := "RGB"
	if sp.Space == gaul.ColorSpaceHSV {
		space = "HSV"
	}
	p.spaceLbl.SetText(space)
}

func (p *SinePreview) CreateRenderer() fyne.WidgetRenderer {
	bold := func(s string) *widget.Label {
		return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	rows := []fyne.CanvasObject{
		p.view,
		widget.NewSeparator(),
		widget.NewLabel("Sine coefficients (read-only):"),
	}
	for i, name := range groupNames {
		rows = append(rows, container.NewBorder(nil, nil, bold(name), nil, p.groupLbls[i]))
	}
	rows = append(rows,
		container.NewBorder(nil, nil, bold("Alpha"), nil, p.alphaLbl),
		container.NewBorder(nil, nil, bold("Color space"), nil, p.spaceLbl),
	)
	return widget.NewSimpleRenderer(container.NewVBox(rows...))
}
