package ui

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
)

var swatchCounts = []int{2, 4, 8, 16, 32, 64}

const stripHeight float32 = 48

// PaletteView is a widget that visualizes any gaul.Palette both as a
// continuous gradient strip and as rows of discrete swatches.
type PaletteView struct {
	widget.BaseWidget
	palette gaul.Palette
}

func NewPaletteView(p gaul.Palette) *PaletteView {
	pv := &PaletteView{palette: p}
	pv.ExtendBaseWidget(pv)
	return pv
}

// SetPalette updates the displayed palette and refreshes.
func (pv *PaletteView) SetPalette(p gaul.Palette) {
	pv.palette = p
	pv.Refresh()
}

func (pv *PaletteView) CreateRenderer() fyne.WidgetRenderer {
	strip := canvas.NewRaster(pv.generateStrip)
	strip.ScaleMode = canvas.ImageScalePixels
	swatchGrid := container.NewVBox()
	placeholder := widget.NewLabel("Select a palette to preview")
	placeholder.Alignment = fyne.TextAlignCenter
	outer := container.NewBorder(strip, nil, nil, nil, swatchGrid)
	r := &paletteViewRenderer{
		view:        pv,
		strip:       strip,
		swatchGrid:  swatchGrid,
		placeholder: placeholder,
		outer:       outer,
	}
	r.buildSwatches()
	return r
}

func (pv *PaletteView) generateStrip(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if w == 0 || pv.palette == nil {
		return img
	}
	for x := 0; x < w; x++ {
		t := float64(x) / float64(w-1)
		c := pv.palette.ColorAt(t)
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	return img
}

type paletteViewRenderer struct {
	view        *PaletteView
	strip       *canvas.Raster
	swatchGrid  *fyne.Container
	placeholder *widget.Label
	outer       *fyne.Container
}

func (r *paletteViewRenderer) buildSwatches() {
	if r.view.palette == nil {
		r.swatchGrid.Objects = nil
		return
	}
	rows := make([]fyne.CanvasObject, len(swatchCounts))
	for ri, n := range swatchCounts {
		cols := make([]fyne.CanvasObject, n)
		for i := 0; i < n; i++ {
			rect := canvas.NewRectangle(r.view.palette.ColorAtStop(i, n))
			rect.SetMinSize(fyne.NewSize(8, 24))
			cols[i] = rect
		}
		rows[ri] = container.NewGridWithColumns(n, cols...)
	}
	r.swatchGrid.Objects = rows
}

func (r *paletteViewRenderer) Layout(size fyne.Size) {
	r.outer.Resize(size)
	r.placeholder.Resize(size)
	r.strip.SetMinSize(fyne.NewSize(size.Width, stripHeight))
}

func (r *paletteViewRenderer) MinSize() fyne.Size {
	return fyne.NewSize(256, stripHeight+float32(len(swatchCounts))*28)
}

func (r *paletteViewRenderer) Refresh() {
	if r.view.palette == nil {
		canvas.Refresh(r.placeholder)
		return
	}
	r.buildSwatches()
	canvas.Refresh(r.outer)
}

func (r *paletteViewRenderer) Objects() []fyne.CanvasObject {
	if r.view.palette == nil {
		return []fyne.CanvasObject{r.placeholder}
	}
	return []fyne.CanvasObject{r.outer}
}

func (r *paletteViewRenderer) Destroy() {}
