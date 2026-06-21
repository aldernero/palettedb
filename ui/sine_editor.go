package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

type vecGroup struct {
	sliders [3]*widget.Slider
	valLbls [3]*widget.Label
	linkBtn *widget.Button
	linked  bool
	syncing bool
}

type SineEditor struct {
	widget.BaseWidget
	database    *db.DB
	window      fyne.Window
	palette     gaul.SinePalette
	view        *PaletteView
	groups      [4]vecGroup
	alphaSlider *widget.Slider
	alphaLbl    *widget.Label
	spaceSelect *widget.Select
	nameEnt     *widget.Entry
	descEnt     *widget.Entry
	OnSaved     func()
}

var (
	groupNames  = [4]string{"A", "B", "C", "D"}
	groupRanges = [4][2]float64{{0, 1}, {0, 1}, {-2, 2}, {-1, 1}}
	axisNames   = [3]string{"X", "Y", "Z"}
)

func NewSineEditor(database *db.DB, window fyne.Window) *SineEditor {
	e := &SineEditor{
		database: database,
		window:   window,
		palette:  gaul.NewSinePalette(gaul.Vec3{X: 1, Y: 0.7, Z: 0.3}, gaul.Vec3{X: 0, Y: 0.15, Z: 0.2}),
		nameEnt:  widget.NewEntry(),
		descEnt:  widget.NewEntry(),
	}
	e.nameEnt.SetPlaceHolder("palette name")
	e.descEnt.SetPlaceHolder("description (optional)")
	e.view = NewPaletteView(&e.palette)
	e.ExtendBaseWidget(e)
	e.initGroups()
	return e
}

func (e *SineEditor) getVec(gi int) *gaul.Vec3 {
	switch gi {
	case 0:
		return &e.palette.A
	case 1:
		return &e.palette.B
	case 2:
		return &e.palette.C
	default:
		return &e.palette.D
	}
}

func getAxis(v *gaul.Vec3, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

func setAxis(v *gaul.Vec3, axis int, val float64) {
	switch axis {
	case 0:
		v.X = val
	case 1:
		v.Y = val
	default:
		v.Z = val
	}
}

func (e *SineEditor) initGroups() {
	for gi := 0; gi < 4; gi++ {
		gi := gi
		g := &e.groups[gi]
		mn, mx := groupRanges[gi][0], groupRanges[gi][1]

		for axis := 0; axis < 3; axis++ {
			axis := axis
			s := widget.NewSlider(mn, mx)
			s.Step = 0.01
			s.Value = getAxis(e.getVec(gi), axis)
			lbl := widget.NewLabel(fmt.Sprintf("%.2f", s.Value))

			s.OnChanged = func(v float64) {
				grp := &e.groups[gi]
				vec := e.getVec(gi)
				setAxis(vec, axis, v)
				grp.valLbls[axis].SetText(fmt.Sprintf("%.2f", v))

				if grp.linked && !grp.syncing {
					grp.syncing = true
					for j := 0; j < 3; j++ {
						if j != axis {
							grp.sliders[j].SetValue(v)
							setAxis(vec, j, v)
							grp.valLbls[j].SetText(fmt.Sprintf("%.2f", v))
						}
					}
					grp.syncing = false
				}
				e.view.SetPalette(&e.palette)
			}

			g.sliders[axis] = s
			g.valLbls[axis] = lbl
		}

		g.linkBtn = widget.NewButton("Link", func() {
			grp := &e.groups[gi]
			grp.linked = !grp.linked
			if grp.linked {
				grp.linkBtn.SetText("Linked")
				grp.linkBtn.Importance = widget.HighImportance
			} else {
				grp.linkBtn.SetText("Link")
				grp.linkBtn.Importance = widget.MediumImportance
			}
			grp.linkBtn.Refresh()
		})
	}

	e.alphaSlider = widget.NewSlider(0, 1)
	e.alphaSlider.Step = 0.01
	e.alphaSlider.Value = e.palette.Alpha
	e.alphaLbl = widget.NewLabel(fmt.Sprintf("%.2f", e.palette.Alpha))
	e.alphaSlider.OnChanged = func(v float64) {
		e.palette.Alpha = v
		e.alphaLbl.SetText(fmt.Sprintf("%.2f", v))
		e.view.SetPalette(&e.palette)
	}

	e.spaceSelect = widget.NewSelect([]string{"RGB", "HSV"}, func(s string) {
		if s == "HSV" {
			e.palette.Space = gaul.ColorSpaceHSV
		} else {
			e.palette.Space = gaul.ColorSpaceRGB
		}
		e.view.SetPalette(&e.palette)
	})
	e.spaceSelect.SetSelected("RGB")
}

func (e *SineEditor) makeGroupWidget(gi int) fyne.CanvasObject {
	g := &e.groups[gi]
	title := widget.NewLabelWithStyle(groupNames[gi], fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil, title, g.linkBtn)

	rows := []fyne.CanvasObject{header}
	for axis := 0; axis < 3; axis++ {
		row := container.NewBorder(nil, nil,
			widget.NewLabel(axisNames[axis]),
			g.valLbls[axis],
			g.sliders[axis],
		)
		rows = append(rows, row)
	}
	return widget.NewCard("", "", container.NewVBox(rows...))
}

func (e *SineEditor) save() {
	name := e.nameEnt.Text
	if name == "" {
		dialog.ShowInformation("Name required", "Please enter a palette name.", e.window)
		return
	}
	_, err := e.database.SaveSine(name, e.descEnt.Text, e.palette)
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

func (e *SineEditor) CreateRenderer() fyne.WidgetRenderer {
	groupGrid := container.NewGridWithColumns(2,
		e.makeGroupWidget(0),
		e.makeGroupWidget(1),
		e.makeGroupWidget(2),
		e.makeGroupWidget(3),
	)

	alphaRow := container.NewBorder(nil, nil,
		widget.NewLabel("Alpha"),
		e.alphaLbl,
		e.alphaSlider,
	)
	spaceRow := container.NewBorder(nil, nil,
		widget.NewLabel("Color space"),
		nil,
		e.spaceSelect,
	)

	nameRow := container.NewGridWithColumns(2, widget.NewLabel("Name:"), e.nameEnt)
	descRow := container.NewGridWithColumns(2, widget.NewLabel("Description:"), e.descEnt)
	saveBtn := widget.NewButton("Save to database", e.save)

	content := container.NewVBox(
		e.view,
		widget.NewSeparator(),
		groupGrid,
		widget.NewSeparator(),
		alphaRow,
		spaceRow,
		widget.NewSeparator(),
		nameRow,
		descRow,
		saveBtn,
	)
	return widget.NewSimpleRenderer(content)
}
