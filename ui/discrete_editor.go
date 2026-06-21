package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

type DiscreteEditor struct {
	widget.BaseWidget
	database *db.DB
	window   fyne.Window
	stops    []color.Color
	view     *PaletteView
	stopList *fyne.Container
	nameEnt  *widget.Entry
	descEnt  *widget.Entry
	OnSaved  func()
}

func NewDiscreteEditor(database *db.DB, window fyne.Window) *DiscreteEditor {
	e := &DiscreteEditor{
		database: database,
		window:   window,
		stops:    []color.Color{color.Black, color.White},
		stopList: container.NewVBox(),
		nameEnt:  widget.NewEntry(),
		descEnt:  widget.NewEntry(),
	}
	e.nameEnt.SetPlaceHolder("palette name")
	e.descEnt.SetPlaceHolder("description (optional)")
	g := e.buildGradient()
	e.view = NewPaletteView(&g)
	e.ExtendBaseWidget(e)
	return e
}

func (e *DiscreteEditor) buildGradient() gaul.Gradient {
	return gaul.NewGradientFromColors(e.stops)
}

func (e *DiscreteEditor) updatePreview() {
	g := e.buildGradient()
	e.view.SetPalette(&g)
}

func (e *DiscreteEditor) rebuildStops() {
	rows := make([]fyne.CanvasObject, len(e.stops))
	for i := range e.stops {
		rows[i] = e.stopRow(i)
	}
	e.stopList.Objects = rows
	e.stopList.Refresh()
	e.updatePreview()
}

func (e *DiscreteEditor) stopRow(i int) fyne.CanvasObject {
	idx := i
	swatch := canvas.NewRectangle(e.stops[idx])
	swatch.SetMinSize(fyne.NewSize(36, 36))

	pickBtn := widget.NewButton("Pick", func() {
		ShowColorPickerDialog("Pick color", e.stops[idx], func(c color.Color) {
			e.stops[idx] = c
			e.rebuildStops()
		}, e.window)
	})

	addBtn := widget.NewButton("+", func() {
		newColor := e.stops[idx]
		e.stops = append(e.stops, nil)
		copy(e.stops[idx+2:], e.stops[idx+1:])
		e.stops[idx+1] = newColor
		e.rebuildStops()
	})

	removeBtn := widget.NewButton("−", func() {
		if len(e.stops) <= 2 {
			return
		}
		e.stops = append(e.stops[:idx], e.stops[idx+1:]...)
		e.rebuildStops()
	})

	return container.NewHBox(swatch, pickBtn, addBtn, removeBtn)
}

func (e *DiscreteEditor) save() {
	name := e.nameEnt.Text
	if name == "" {
		dialog.ShowInformation("Name required", "Please enter a palette name.", e.window)
		return
	}
	_, err := e.database.SaveDiscrete(name, e.descEnt.Text, e.stops)
	if err != nil {
		dialog.ShowError(err, e.window)
		return
	}
	if e.OnSaved != nil {
		e.OnSaved()
	}
	e.nameEnt.SetText("")
	e.descEnt.SetText("")
}

func (e *DiscreteEditor) CreateRenderer() fyne.WidgetRenderer {
	e.rebuildStops()

	addBtn := widget.NewButton("Add stop", func() {
		e.stops = append(e.stops, color.White)
		e.rebuildStops()
	})

	nameRow := container.NewGridWithColumns(2, widget.NewLabel("Name:"), e.nameEnt)
	descRow := container.NewGridWithColumns(2, widget.NewLabel("Description:"), e.descEnt)
	saveBtn := widget.NewButton("Save to database", e.save)

	content := container.NewVBox(
		e.view,
		widget.NewSeparator(),
		widget.NewLabel("Color stops:"),
		e.stopList,
		addBtn,
		widget.NewSeparator(),
		nameRow,
		descRow,
		saveBtn,
	)
	return widget.NewSimpleRenderer(content)
}
