package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/colornames"
)

// ---- math helpers ----

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func pickerHSVtoRGB(h, s, v float64) (r, g, b float64) {
	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1.0
	}
	if s == 0 {
		return v, v, v
	}
	h6 := h * 6
	i := int(h6)
	f := h6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	u := v * (1 - s*(1-f))
	switch i % 6 {
	case 0:
		return v, u, p
	case 1:
		return q, v, p
	case 2:
		return p, v, u
	case 3:
		return p, q, v
	case 4:
		return u, p, v
	default:
		return v, p, q
	}
}

func pickerRGBtoHSV(r, g, b float64) (h, s, v float64) {
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	v = mx
	d := mx - mn
	if mx == 0 {
		return 0, 0, v
	}
	s = d / mx
	if d == 0 {
		return 0, s, v
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

func colorToPickerHSV(c color.Color) (h, s, v float64) {
	r32, g32, b32, _ := c.RGBA()
	return pickerRGBtoHSV(float64(r32)/65535, float64(g32)/65535, float64(b32)/65535)
}

// ---- interactiveRaster widget ----

// interactiveRaster is a widget that renders an image and forwards click/drag
// positions as relative [0,1] coordinates via onPos.
type interactiveRaster struct {
	widget.BaseWidget
	gen    func(w, h int) image.Image
	onPos  func(relX, relY float64)
	raster *canvas.Raster
	minW   float32
	minH   float32
}

func newInteractiveRaster(gen func(w, h int) image.Image, onPos func(float64, float64), minW, minH float32) *interactiveRaster {
	r := &interactiveRaster{gen: gen, onPos: onPos, minW: minW, minH: minH}
	r.raster = canvas.NewRaster(gen)
	r.raster.ScaleMode = canvas.ImageScalePixels
	r.ExtendBaseWidget(r)
	return r
}

func (r *interactiveRaster) CreateRenderer() fyne.WidgetRenderer {
	return &interactiveRasterRenderer{r: r}
}

func (r *interactiveRaster) handlePos(p fyne.Position) {
	sz := r.Size()
	if sz.Width == 0 || sz.Height == 0 {
		return
	}
	r.onPos(
		clamp01(float64(p.X)/float64(sz.Width)),
		clamp01(float64(p.Y)/float64(sz.Height)),
	)
}

func (r *interactiveRaster) Tapped(e *fyne.PointEvent) { r.handlePos(e.Position) }
func (r *interactiveRaster) Dragged(e *fyne.DragEvent) { r.handlePos(e.Position) }
func (r *interactiveRaster) DragEnd()                   {}

type interactiveRasterRenderer struct{ r *interactiveRaster }

func (rr *interactiveRasterRenderer) Layout(sz fyne.Size) {
	rr.r.raster.Move(fyne.NewPos(0, 0))
	rr.r.raster.Resize(sz)
}
func (rr *interactiveRasterRenderer) MinSize() fyne.Size {
	return fyne.NewSize(rr.r.minW, rr.r.minH)
}
func (rr *interactiveRasterRenderer) Refresh()                        { rr.r.raster.Refresh() }
func (rr *interactiveRasterRenderer) Objects() []fyne.CanvasObject    { return []fyne.CanvasObject{rr.r.raster} }
func (rr *interactiveRasterRenderer) Destroy()                        {}

// ---- picker state ----

type pickerState struct {
	h, s, v float64
	syncing  bool

	hueRaster *interactiveRaster
	svRaster  *interactiveRaster
	preview   *canvas.Rectangle

	hEnt, sEnt, vEnt *widget.Entry
	rEnt, gEnt, bEnt *widget.Entry
	hexEnt            *widget.Entry
	namedSel *widget.Select
	allNames []string
}

func drawHueCursor(img *image.RGBA, cx, h int) {
	w := img.Bounds().Max.X
	for y := 0; y < h; y++ {
		if cx > 0 {
			img.Set(cx-1, y, color.Black)
		}
		img.Set(cx, y, color.White)
		if cx < w-1 {
			img.Set(cx+1, y, color.Black)
		}
	}
}

func drawSVCursor(img *image.RGBA, cx, cy int) {
	b := img.Bounds()
	const r = 6
	for dx := -(r + 2); dx <= r+2; dx++ {
		for dy := -(r + 2); dy <= r+2; dy++ {
			x, y := cx+dx, cy+dy
			if x < 0 || x >= b.Max.X || y < 0 || y >= b.Max.Y {
				continue
			}
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			switch {
			case dist >= r-0.5 && dist <= r+0.5:
				img.Set(x, y, color.White)
			case dist > r+0.5 && dist <= r+1.5:
				img.Set(x, y, color.Black)
			}
		}
	}
}

func (ps *pickerState) genHueBar(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		r, g, b := pickerHSVtoRGB(float64(x)/float64(w), 1, 1)
		c := color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	drawHueCursor(img, int(ps.h*float64(w)), h)
	return img
}

func (ps *pickerState) genSVSquare(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		sat := float64(x) / float64(w)
		for y := 0; y < h; y++ {
			val := 1.0 - float64(y)/float64(h)
			r, g, b := pickerHSVtoRGB(ps.h, sat, val)
			img.Set(x, y, color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255})
		}
	}
	drawSVCursor(img, int(ps.s*float64(w)), int((1-ps.v)*float64(h)))
	return img
}

func (ps *pickerState) setState(h, s, v float64) {
	if ps.syncing {
		return
	}
	ps.syncing = true
	defer func() { ps.syncing = false }()

	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1.0
	}
	ps.h = h
	ps.s = clamp01(s)
	ps.v = clamp01(v)

	r, g, b := pickerHSVtoRGB(ps.h, ps.s, ps.v)
	ri := int(math.Round(r * 255))
	gi := int(math.Round(g * 255))
	bi := int(math.Round(b * 255))

	ps.hEnt.SetText(fmt.Sprintf("%.1f", ps.h*360))
	ps.sEnt.SetText(fmt.Sprintf("%.1f", ps.s*100))
	ps.vEnt.SetText(fmt.Sprintf("%.1f", ps.v*100))
	ps.rEnt.SetText(fmt.Sprintf("%d", ri))
	ps.gEnt.SetText(fmt.Sprintf("%d", gi))
	ps.bEnt.SetText(fmt.Sprintf("%d", bi))
	ps.hexEnt.SetText(fmt.Sprintf("#%02X%02X%02X", ri, gi, bi))

	ps.preview.FillColor = color.RGBA{R: uint8(ri), G: uint8(gi), B: uint8(bi), A: 255}
	canvas.Refresh(ps.preview)
	ps.hueRaster.Refresh()
	ps.svRaster.Refresh()
}

func (ps *pickerState) currentColor() color.Color {
	r, g, b := pickerHSVtoRGB(ps.h, ps.s, ps.v)
	return color.RGBA{
		R: uint8(math.Round(r * 255)),
		G: uint8(math.Round(g * 255)),
		B: uint8(math.Round(b * 255)),
		A: 255,
	}
}

func (ps *pickerState) buildUI() fyne.CanvasObject {
	// named color list
	ps.allNames = make([]string, 0, len(colornames.Map))
	for name := range colornames.Map {
		ps.allNames = append(ps.allNames, name)
	}
	sort.Strings(ps.allNames)

	// rasters
	ps.hueRaster = newInteractiveRaster(ps.genHueBar, func(relX, _ float64) {
		ps.setState(relX, ps.s, ps.v)
	}, 200, 22)

	ps.svRaster = newInteractiveRaster(ps.genSVSquare, func(relX, relY float64) {
		ps.setState(ps.h, relX, 1-relY)
	}, 200, 220)

	// preview swatch
	r0, g0, b0 := pickerHSVtoRGB(ps.h, ps.s, ps.v)
	ri0 := int(math.Round(r0 * 255))
	gi0 := int(math.Round(g0 * 255))
	bi0 := int(math.Round(b0 * 255))
	ps.preview = canvas.NewRectangle(color.RGBA{uint8(ri0), uint8(gi0), uint8(bi0), 255})
	ps.preview.SetMinSize(fyne.NewSize(200, 32))

	// HSV entries (H in degrees 0–360, S and V in percent 0–100)
	ps.hEnt = widget.NewEntry()
	ps.sEnt = widget.NewEntry()
	ps.vEnt = widget.NewEntry()
	ps.hEnt.SetText(fmt.Sprintf("%.1f", ps.h*360))
	ps.sEnt.SetText(fmt.Sprintf("%.1f", ps.s*100))
	ps.vEnt.SetText(fmt.Sprintf("%.1f", ps.v*100))

	ps.hEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		deg, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return
		}
		ps.setState(deg/360, ps.s, ps.v)
	}
	ps.sEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		pct, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return
		}
		ps.setState(ps.h, pct/100, ps.v)
	}
	ps.vEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		pct, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return
		}
		ps.setState(ps.h, ps.s, pct/100)
	}

	// RGB entries (0–255 integers)
	ps.rEnt = widget.NewEntry()
	ps.gEnt = widget.NewEntry()
	ps.bEnt = widget.NewEntry()
	ps.rEnt.SetText(fmt.Sprintf("%d", ri0))
	ps.gEnt.SetText(fmt.Sprintf("%d", gi0))
	ps.bEnt.SetText(fmt.Sprintf("%d", bi0))

	ps.rEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return
		}
		_, cg, cb := pickerHSVtoRGB(ps.h, ps.s, ps.v)
		h, s, v := pickerRGBtoHSV(float64(n)/255, cg, cb)
		ps.setState(h, s, v)
	}
	ps.gEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return
		}
		cr, _, cb := pickerHSVtoRGB(ps.h, ps.s, ps.v)
		h, s, v := pickerRGBtoHSV(cr, float64(n)/255, cb)
		ps.setState(h, s, v)
	}
	ps.bEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return
		}
		cr, cg, _ := pickerHSVtoRGB(ps.h, ps.s, ps.v)
		h, s, v := pickerRGBtoHSV(cr, cg, float64(n)/255)
		ps.setState(h, s, v)
	}

	// Hex entry
	ps.hexEnt = widget.NewEntry()
	ps.hexEnt.SetText(fmt.Sprintf("#%02X%02X%02X", ri0, gi0, bi0))
	ps.hexEnt.OnChanged = func(t string) {
		if ps.syncing {
			return
		}
		hex := strings.ToUpper(strings.TrimPrefix(t, "#"))
		if len(hex) != 6 {
			return
		}
		rv, e1 := strconv.ParseUint(hex[0:2], 16, 8)
		gv, e2 := strconv.ParseUint(hex[2:4], 16, 8)
		bv, e3 := strconv.ParseUint(hex[4:6], 16, 8)
		if e1 != nil || e2 != nil || e3 != nil {
			return
		}
		h, s, v := pickerRGBtoHSV(float64(rv)/255, float64(gv)/255, float64(bv)/255)
		ps.setState(h, s, v)
	}

	// Named color selector with search
	ps.namedSel = widget.NewSelect(ps.allNames, func(name string) {
		if ps.syncing {
			return
		}
		c, ok := colornames.Map[name]
		if !ok {
			return
		}
		h, s, v := pickerRGBtoHSV(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255)
		ps.setState(h, s, v)
	})

	hsvRow := container.NewGridWithColumns(6,
		widget.NewLabelWithStyle("H°", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.hEnt,
		widget.NewLabelWithStyle("S%", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.sEnt,
		widget.NewLabelWithStyle("V%", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.vEnt,
	)
	rgbRow := container.NewGridWithColumns(6,
		widget.NewLabelWithStyle("R", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.rEnt,
		widget.NewLabelWithStyle("G", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.gEnt,
		widget.NewLabelWithStyle("B", fyne.TextAlignTrailing, fyne.TextStyle{}), ps.bEnt,
	)
	hexRow := container.NewBorder(nil, nil, widget.NewLabel("Hex"), nil, ps.hexEnt)

	return container.NewVBox(
		ps.hueRaster,
		ps.svRaster,
		ps.preview,
		widget.NewSeparator(),
		hsvRow,
		rgbRow,
		hexRow,
		widget.NewSeparator(),
		ps.namedSel,
	)
}

// ShowColorPickerDialog opens a custom HSV color picker dialog.
// initial is shown as the starting color; onPick is called with the chosen
// color when the user clicks OK.
func ShowColorPickerDialog(title string, initial color.Color, onPick func(color.Color), w fyne.Window) {
	ps := &pickerState{}
	if initial != nil {
		ps.h, ps.s, ps.v = colorToPickerHSV(initial)
	}
	content := ps.buildUI()
	d := dialog.NewCustomConfirm(title, "OK", "Cancel", content, func(ok bool) {
		if ok && onPick != nil {
			onPick(ps.currentColor())
		}
	}, w)
	d.Resize(fyne.NewSize(360, 640))
	d.Show()
}
