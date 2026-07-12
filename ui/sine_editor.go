package ui

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/ui/resources"
)

// vecGroup is one of the A/B/C/D coefficient vectors of a sine palette. Each
// axis (X/Y/Z) can be locked: a locked axis is held — excluded from both the
// link movement and the dice/shuffle. When the group is linked, dragging any
// unlocked axis moves all the other unlocked axes by the same delta (their
// relative offsets are preserved), clamped as a group so none leaves its range.
type vecGroup struct {
	sliders  [3]*widget.Slider
	entries  [3]*numericEntry
	locks    [3]bool
	lockBtns [3]*widget.Button
	linkBtn  *widget.Button
	linked   bool
	diceBtn  *widget.Button
	syncing  bool
}

// SineEditor edits a gaul.SinePalette. Name/description/saving are handled by
// the Workspace controller.
type SineEditor struct {
	widget.BaseWidget
	window      fyne.Window
	palette     gaul.SinePalette
	view        *PaletteView
	groups      [4]vecGroup
	alphaSlider *widget.Slider
	alphaLbl    *widget.Label
	spaceSelect *widget.Select
	OnChange    func()
}

var (
	groupNames  = [4]string{"A", "B", "C", "D"}
	groupRanges = [4][2]float64{{-7, 7}, {-7, 7}, {0, 7}, {0, 1}}
	axisNames   = [3]string{"X", "Y", "Z"}
)

func defaultSinePalette() gaul.SinePalette {
	return gaul.NewSinePalette(gaul.Vec3{X: 1, Y: 0.7, Z: 0.3}, gaul.Vec3{X: 0, Y: 0.15, Z: 0.2})
}

func NewSineEditor(window fyne.Window) *SineEditor {
	e := &SineEditor{window: window, palette: defaultSinePalette()}
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

// setAxisRaw writes one axis to the model, slider, and entry without any link
// propagation. Must be called with grp.syncing already true.
func (e *SineEditor) setAxisRaw(grp *vecGroup, vec *gaul.Vec3, axis int, v float64) {
	setAxis(vec, axis, v)
	grp.sliders[axis].SetValue(v)
	grp.entries[axis].SetText(fmt.Sprintf("%.3f", v))
}

// applyAxisChange applies a requested new value to one axis. When the group is
// linked and the axis is unlocked, all other unlocked axes move by the same
// delta (offsets preserved), clamped as a group so no unlocked axis leaves its
// range. Otherwise only the one axis changes.
func (e *SineEditor) applyAxisChange(gi, axis int, v float64) {
	grp := &e.groups[gi]
	vec := e.getVec(gi)
	mn, mx := groupRanges[gi][0], groupRanges[gi][1]
	v = gaul.Clamp(mn, mx, v)

	grp.syncing = true
	if grp.linked && !grp.locks[axis] {
		delta := v - getAxis(vec, axis)
		// Limit delta so every unlocked axis stays within [mn, mx].
		for j := 0; j < 3; j++ {
			if grp.locks[j] {
				continue
			}
			cur := getAxis(vec, j)
			if delta > 0 && cur+delta > mx {
				delta = mx - cur
			} else if delta < 0 && cur+delta < mn {
				delta = mn - cur
			}
		}
		for j := 0; j < 3; j++ {
			if grp.locks[j] {
				continue
			}
			e.setAxisRaw(grp, vec, j, getAxis(vec, j)+delta)
		}
	} else {
		e.setAxisRaw(grp, vec, axis, v)
	}
	grp.syncing = false
	e.refreshPreview()
}

func (e *SineEditor) refreshPreview() {
	e.view.SetPalette(&e.palette)
	if e.OnChange != nil {
		e.OnChange()
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
			ent := newNumericEntry()
			ent.SetText(fmt.Sprintf("%.3f", s.Value))

			s.OnChanged = func(v float64) {
				if e.groups[gi].syncing {
					return
				}
				e.applyAxisChange(gi, axis, v)
			}
			// Commit on Enter and on focus loss (see numericEntry).
			ent.onCommit = func(text string) {
				v, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
				if err != nil {
					// Restore the current value on bad input.
					e.groups[gi].entries[axis].SetText(fmt.Sprintf("%.3f", getAxis(e.getVec(gi), axis)))
					return
				}
				if v == getAxis(e.getVec(gi), axis) {
					return // unchanged; avoid a spurious refresh/dirty on blur
				}
				e.applyAxisChange(gi, axis, v)
			}

			lockBtn := widget.NewButtonWithIcon("", resources.UnlockIcon, nil)
			lockBtn.Importance = widget.LowImportance
			lockBtn.OnTapped = func() {
				grp := &e.groups[gi]
				grp.locks[axis] = !grp.locks[axis]
				if grp.locks[axis] {
					grp.lockBtns[axis].SetIcon(resources.LockIcon)
				} else {
					grp.lockBtns[axis].SetIcon(resources.UnlockIcon)
				}
			}

			g.sliders[axis] = s
			g.entries[axis] = ent
			g.lockBtns[axis] = lockBtn
		}

		g.linkBtn = widget.NewButtonWithIcon("", resources.LinkOffIcon, nil)
		g.linkBtn.Importance = widget.LowImportance
		g.linkBtn.OnTapped = func() {
			grp := &e.groups[gi]
			grp.linked = !grp.linked
			if grp.linked {
				grp.linkBtn.SetIcon(resources.LinkIcon)
				grp.linkBtn.Importance = widget.HighImportance
			} else {
				grp.linkBtn.SetIcon(resources.LinkOffIcon)
				grp.linkBtn.Importance = widget.LowImportance
			}
			grp.linkBtn.Refresh()
		}

		g.diceBtn = widget.NewButtonWithIcon("", resources.DiceIcon, func() {
			e.shuffleGroup(gi)
		})
		g.diceBtn.Importance = widget.LowImportance
	}

	e.alphaSlider = widget.NewSlider(0, 1)
	e.alphaSlider.Step = 0.01
	e.alphaSlider.Value = e.palette.Alpha
	e.alphaLbl = widget.NewLabel(fmt.Sprintf("%.3f", e.palette.Alpha))
	e.alphaSlider.OnChanged = func(v float64) {
		e.palette.Alpha = v
		e.alphaLbl.SetText(fmt.Sprintf("%.3f", v))
		e.refreshPreview()
	}

	e.spaceSelect = widget.NewSelect([]string{"RGB", "HSV"}, func(s string) {
		if s == "HSV" {
			e.palette.Space = gaul.ColorSpaceHSV
		} else {
			e.palette.Space = gaul.ColorSpaceRGB
		}
		e.refreshPreview()
	})
	e.spaceSelect.SetSelected("RGB")
}

// shuffleGroup randomizes the unlocked axes of a group within its range. Each
// unlocked axis is randomized independently (link movement does not apply).
func (e *SineEditor) shuffleGroup(gi int) {
	mn, mx := groupRanges[gi][0], groupRanges[gi][1]
	grp := &e.groups[gi]
	vec := e.getVec(gi)
	grp.syncing = true
	for axis := 0; axis < 3; axis++ {
		if grp.locks[axis] {
			continue
		}
		e.setAxisRaw(grp, vec, axis, mn+rand.Float64()*(mx-mn))
	}
	grp.syncing = false
	e.refreshPreview()
}

func (e *SineEditor) makeGroupWidget(gi int) fyne.CanvasObject {
	g := &e.groups[gi]
	title := widget.NewLabelWithStyle(groupNames[gi], fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil, title, container.NewHBox(g.linkBtn, g.diceBtn))

	rows := []fyne.CanvasObject{header}
	for axis := 0; axis < 3; axis++ {
		left := container.NewHBox(g.lockBtns[axis], widget.NewLabel(axisNames[axis]))
		entry := container.NewGridWrap(fyne.NewSize(66, g.entries[axis].MinSize().Height), g.entries[axis])
		row := container.NewBorder(nil, nil, left, entry, g.sliders[axis])
		rows = append(rows, row)
	}
	return widget.NewCard("", "", container.NewVBox(rows...))
}

// Palette returns the current sine palette.
func (e *SineEditor) Palette() gaul.SinePalette { return e.palette }

// LoadPalette populates the editor from an existing sine palette.
func (e *SineEditor) LoadPalette(sp gaul.SinePalette) {
	e.palette = sp
	for gi := 0; gi < 4; gi++ {
		grp := &e.groups[gi]
		vec := e.getVec(gi)
		grp.syncing = true
		for axis := 0; axis < 3; axis++ {
			v := getAxis(vec, axis)
			grp.sliders[axis].SetValue(v)
			grp.entries[axis].SetText(fmt.Sprintf("%.3f", v))
			// Reset lock state for a clean slate on the new document.
			grp.locks[axis] = false
			grp.lockBtns[axis].SetIcon(resources.UnlockIcon)
		}
		grp.linked = false
		grp.linkBtn.SetIcon(resources.LinkOffIcon)
		grp.linkBtn.Importance = widget.LowImportance
		grp.linkBtn.Refresh()
		grp.syncing = false
	}
	e.alphaSlider.SetValue(sp.Alpha)
	e.alphaLbl.SetText(fmt.Sprintf("%.3f", sp.Alpha))
	if sp.Space == gaul.ColorSpaceHSV {
		e.spaceSelect.SetSelected("HSV")
	} else {
		e.spaceSelect.SetSelected("RGB")
	}
	e.refreshPreview()
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

	content := container.NewVBox(
		e.view,
		widget.NewSeparator(),
		groupGrid,
		widget.NewSeparator(),
		alphaRow,
		spaceRow,
	)
	return widget.NewSimpleRenderer(content)
}
