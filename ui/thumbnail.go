package ui

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
)

// thumbnailSize is the fixed size of a palette list thumbnail.
var thumbnailSize = fyne.NewSize(52, 18)

// PaletteThumbnail is a small gradient-strip preview of a palette, used as the
// icon in the browse list. It renders only the continuous gradient (no swatch
// grid), keeping it compact.
type PaletteThumbnail struct {
	widget.BaseWidget
	palette gaul.Palette
}

func NewPaletteThumbnail(p gaul.Palette) *PaletteThumbnail {
	t := &PaletteThumbnail{palette: p}
	t.ExtendBaseWidget(t)
	return t
}

// SetPalette updates the previewed palette and refreshes.
func (t *PaletteThumbnail) SetPalette(p gaul.Palette) {
	t.palette = p
	t.Refresh()
}

func (t *PaletteThumbnail) CreateRenderer() fyne.WidgetRenderer {
	raster := canvas.NewRaster(t.generate)
	raster.ScaleMode = canvas.ImageScalePixels
	raster.SetMinSize(thumbnailSize)
	return widget.NewSimpleRenderer(raster)
}

func (t *PaletteThumbnail) generate(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if w == 0 || t.palette == nil {
		return img
	}
	for x := 0; x < w; x++ {
		tt := 0.0
		if w > 1 {
			tt = float64(x) / float64(w-1)
		}
		c := t.palette.ColorAt(tt)
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	return img
}
