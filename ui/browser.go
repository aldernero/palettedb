package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/palettedb/internal/db"
)

// Browser shows all saved palettes and lets the user inspect or delete them.
type Browser struct {
	widget.BaseWidget
	database   *db.DB
	window     fyne.Window
	entries    []db.Entry
	list       *widget.List
	view       *PaletteView
	infoLbl    *widget.Label
	selectedID int64 // directory ID of selected entry, -1 if none
}

func NewBrowser(database *db.DB, window fyne.Window) *Browser {
	b := &Browser{
		database:   database,
		window:     window,
		view:       NewPaletteView(nil),
		infoLbl:    widget.NewLabel(""),
		selectedID: -1,
	}
	b.ExtendBaseWidget(b)
	b.list = widget.NewList(
		func() int { return len(b.entries) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(fmt.Sprintf("%s  [%s]", b.entries[id].Name, b.entries[id].Type))
		},
	)
	b.list.OnSelected = b.onSelect
	b.Refresh()
	return b
}

func (b *Browser) refresh() {
	entries, err := b.database.ListAll()
	if err != nil {
		b.infoLbl.SetText("Error: " + err.Error())
		return
	}
	b.entries = entries
	b.list.Refresh()
}

func (b *Browser) onSelect(id widget.ListItemID) {
	if id < 0 || id >= len(b.entries) {
		b.selectedID = -1
		return
	}
	e := b.entries[id]
	b.selectedID = e.ID
	switch e.Type {
	case "sine":
		r, err := b.database.LoadSine(e.ID)
		if err != nil {
			b.infoLbl.SetText("Load error: " + err.Error())
			return
		}
		b.view.SetPalette(&r.Palette)
	case "discrete":
		r, err := b.database.LoadDiscrete(e.ID)
		if err != nil {
			b.infoLbl.SetText("Load error: " + err.Error())
			return
		}
		g := r.Gradient()
		b.view.SetPalette(&g)
	}
	b.infoLbl.SetText(fmt.Sprintf("%s  |  type: %s  |  %s", e.Name, e.Type, e.Description))
}

func (b *Browser) deleteSelected() {
	if b.selectedID < 0 {
		return
	}
	id := b.selectedID
	name := ""
	for _, e := range b.entries {
		if e.ID == id {
			name = e.Name
			break
		}
	}
	dialog.ShowConfirm("Delete palette", fmt.Sprintf("Delete %q?", name), func(ok bool) {
		if !ok {
			return
		}
		if err := b.database.Delete(id); err != nil {
			dialog.ShowError(err, b.window)
			return
		}
		b.selectedID = -1
		b.refresh()
	}, b.window)
}

func (b *Browser) CreateRenderer() fyne.WidgetRenderer {
	b.refresh()
	deleteBtn := widget.NewButton("Delete", b.deleteSelected)
	left := container.NewBorder(nil, deleteBtn, nil, nil, container.NewScroll(b.list))
	right := container.NewBorder(b.infoLbl, nil, nil, nil, b.view)
	split := container.NewHSplit(left, right)
	split.SetOffset(0.28)
	return widget.NewSimpleRenderer(split)
}
