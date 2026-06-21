package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"github.com/aldernero/palettedb/internal/db"
)

func Run() {
	a := app.New()
	w := a.NewWindow("PaletteDB")
	w.Resize(fyne.NewSize(960, 720))

	database, err := db.Open()
	if err != nil {
		dialog.ShowError(err, w)
	}

	browser := NewBrowser(database, w)
	discreteEd := NewDiscreteEditor(database, w)
	sineEd := NewSineEditor(database, w)

	discreteEd.OnSaved = browser.refresh
	sineEd.OnSaved = browser.refresh

	tabs := container.NewAppTabs(
		container.NewTabItem("Browse", browser),
		container.NewTabItem("New Discrete", container.NewScroll(discreteEd)),
		container.NewTabItem("New Sine", container.NewScroll(sineEd)),
	)

	w.SetContent(tabs)
	w.ShowAndRun()
}
