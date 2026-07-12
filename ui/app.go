package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/builtins"
	"github.com/aldernero/palettedb/internal/db"
	"github.com/aldernero/palettedb/internal/export"
	"github.com/aldernero/palettedb/ui/resources"
)

// initialWindowSize is the window size on launch.
var initialWindowSize = fyne.NewSize(1000, 760)

const appID = "com.aldernero.palettedb"
const appVersion = "0.1.0"

type docKind int

const (
	docNone docKind = iota
	docSine
	docDiscrete
)

// newDocIDBase is the (very negative) base for synthetic ids of unsaved new
// documents, chosen so they never collide with DB ids (>0) or built-in ids
// (small negatives).
const newDocIDBase = -1_000_000

// document is an in-memory palette: either a new unsaved one or an existing one
// that has been opened/modified. Its content lives here until saved to the DB.
type document struct {
	id    int64 // DB id (>0) once saved; synthetic (<= newDocIDBase) while new
	kind  docKind
	name  string
	desc  string
	dirty bool // has unsaved changes
	saved bool // exists in the DB
	sine  gaul.SinePalette
	stops []db.ColorStop
}

// Workspace is the top-level controller: it owns the window, the browse sidebar,
// the editors/previews, and the set of open/modified documents.
type Workspace struct {
	app        fyne.App
	win        fyne.Window
	db         *db.DB
	browser    *Browser
	sineEd     *SineEditor
	discreteEd *DiscreteEditor

	discretePreview *DiscretePreview
	sinePreview     *SinePreview
	welcome         fyne.CanvasObject

	editorHost     *fyne.Container
	sineScroll     *container.Scroll
	discreteScroll *container.Scroll
	header         *widget.Label

	themeChoice string
	loading     bool // suppress dirty marking during programmatic loads

	// Document model.
	docs        map[int64]*document // new + dirty documents (by id)
	current     *document           // custom palette being edited (nil otherwise)
	newSeq      int64               // decrementing id source for new documents
	untitledSeq int                 // "Untitled #N" counter

	// Built-in preview state (current == nil, readOnly == true).
	readOnly     bool
	builtinName  string
	builtinKind  docKind
	builtinStops []db.ColorStop
	builtinSine  gaul.SinePalette
}

func kindString(k docKind) string {
	if k == docSine {
		return "sine"
	}
	return "discrete"
}

func defaultStops() []db.ColorStop {
	return []db.ColorStop{{Color: color.Black, Pos: 0}, {Color: color.White, Pos: 1}}
}

// Run builds and runs the palettedb application.
func Run() {
	a := app.NewWithID(appID)
	a.SetIcon(resources.AppIcon)
	w := a.NewWindow("PaletteDB")
	w.SetIcon(resources.AppIcon)
	w.Resize(initialWindowSize)

	database, err := db.Open()
	if err != nil {
		dialog.ShowError(err, w)
	}

	ws := &Workspace{app: a, win: w, db: database}
	ws.themeChoice = a.Preferences().StringWithFallback(themePrefKey, themeSystem)
	applyThemeChoice(a, ws.themeChoice)

	ws.build()
	ws.showWelcome()

	w.SetMainMenu(ws.buildMenu())

	// platformWindowInit / platformAfterShow differ by backend (see app_x11.go
	// and app_wayland.go): the X11/XWayland build needs a relayout nudge to fix
	// initial HiDPI scaling; the native Wayland build does not.
	platformWindowInit(w)
	w.Show()
	platformAfterShow(w)
	a.Run()
}

func (w *Workspace) build() {
	w.docs = make(map[int64]*document)
	w.newSeq = newDocIDBase

	w.browser = NewBrowser(w.db, w.win)
	w.browser.OnSelect = w.onBrowseSelect
	w.browser.OnContext = w.showContextMenu

	w.sineEd = NewSineEditor(w.win)
	w.discreteEd = NewDiscreteEditor(w.win)
	w.sineEd.OnChange = w.onEditorChanged
	w.discreteEd.OnChange = w.onEditorChanged

	w.discretePreview = NewDiscretePreview()
	w.sinePreview = NewSinePreview()
	w.welcome = w.buildWelcome()

	w.sineScroll = container.NewScroll(w.sineEd)
	w.discreteScroll = container.NewScroll(w.discreteEd)

	w.header = widget.NewLabel("")
	w.header.TextStyle = fyne.TextStyle{Bold: true}
	w.editorHost = container.NewStack()

	editorArea := container.NewBorder(w.header, nil, nil, nil, w.editorHost)
	split := container.NewHSplit(w.browser, editorArea)
	split.SetOffset(0.26)
	w.win.SetContent(split)

	w.refreshBrowser()
	w.win.SetCloseIntercept(w.onCloseRequested)
}

func (w *Workspace) onEditorChanged() {
	if w.loading || w.current == nil {
		return
	}
	if !w.current.dirty {
		w.current.dirty = true
		w.docs[w.current.id] = w.current
		w.updateHeader()
		w.refreshBrowser()
	}
}

// showEditor swaps the visible editor (or the read-only preview for built-ins,
// or the welcome screen when nothing is open).
func (w *Workspace) showEditor(kind docKind) {
	switch {
	case w.readOnly && kind == docDiscrete:
		// Shown directly (not scrolled) so the stops table fills the space.
		w.editorHost.Objects = []fyne.CanvasObject{w.discretePreview}
	case w.readOnly && kind == docSine:
		w.editorHost.Objects = []fyne.CanvasObject{container.NewScroll(w.sinePreview)}
	case kind == docSine:
		w.editorHost.Objects = []fyne.CanvasObject{w.sineScroll}
	case kind == docDiscrete:
		w.editorHost.Objects = []fyne.CanvasObject{w.discreteScroll}
	default:
		w.editorHost.Objects = []fyne.CanvasObject{w.welcome}
	}
	w.editorHost.Refresh()
}

// showWelcome clears the current document and shows the start screen.
func (w *Workspace) showWelcome() {
	w.captureCurrent()
	w.current = nil
	w.readOnly = false
	w.browser.UnselectAll()
	w.showEditor(docNone)
	w.updateHeader()
}

// buildWelcome constructs the start screen shown when no palette is open.
func (w *Workspace) buildWelcome() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(resources.AppIcon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(120, 120))

	title := widget.NewLabelWithStyle("Welcome to PaletteDB", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	hint := widget.NewLabelWithStyle(
		"Select a palette from the list to view or edit it,\nor create a new one below.",
		fyne.TextAlignCenter, fyne.TextStyle{})
	newSineBtn := widget.NewButtonWithIcon("New Sine Palette", resources.SineIcon, w.newSine)
	newDiscBtn := widget.NewButtonWithIcon("New Discrete Palette", resources.DiscreteIcon, w.newDiscrete)

	col := container.NewVBox(
		container.NewCenter(logo),
		title,
		hint,
		container.NewCenter(container.NewHBox(newSineBtn, newDiscBtn)),
	)
	return container.NewCenter(col)
}

// captureCurrent pulls the live editor state into the current document.
func (w *Workspace) captureCurrent() {
	if w.current == nil {
		return
	}
	switch w.current.kind {
	case docSine:
		w.current.sine = w.sineEd.Palette()
	case docDiscrete:
		w.current.stops = w.discreteEd.Stops()
	}
}

// loadDoc shows a custom document in the matching editor.
func (w *Workspace) loadDoc(d *document) {
	w.readOnly = false
	w.current = d
	w.loading = true
	switch d.kind {
	case docSine:
		w.sineEd.LoadPalette(d.sine)
	case docDiscrete:
		w.discreteEd.LoadStops(d.stops)
	}
	w.loading = false
	w.showEditor(d.kind)
	w.updateHeader()
}

func (w *Workspace) updateHeader() {
	switch {
	case w.current != nil:
		marker := ""
		if w.current.dirty {
			marker = " *"
		}
		w.header.SetText(fmt.Sprintf("%s%s  (%s)", w.current.name, marker, kindString(w.current.kind)))
		w.win.SetTitle(fmt.Sprintf("PaletteDB — %s%s", w.current.name, marker))
	case w.readOnly:
		w.header.SetText(fmt.Sprintf("%s  (built-in, read-only)", w.builtinName))
		w.win.SetTitle("PaletteDB — " + w.builtinName)
	default:
		w.header.SetText("")
		w.win.SetTitle("PaletteDB")
	}
}

// --- New / Open / Copy ---

func (w *Workspace) newSine()     { w.newDoc(docSine) }
func (w *Workspace) newDiscrete() { w.newDoc(docDiscrete) }

// newDoc creates a new unsaved document and opens it in the editor.
func (w *Workspace) newDoc(kind docKind) {
	w.captureCurrent()
	id := w.newSeq
	w.newSeq--
	d := &document{id: id, kind: kind, name: w.nextUntitledName(), dirty: true}
	switch kind {
	case docSine:
		d.sine = defaultSinePalette()
	case docDiscrete:
		d.stops = defaultStops()
	}
	w.docs[id] = d
	w.loadDoc(d)
	w.refreshBrowser()
	w.browser.SelectByID(id)
}

func (w *Workspace) nextUntitledName() string {
	for {
		w.untitledSeq++
		name := fmt.Sprintf("Untitled #%d", w.untitledSeq)
		if !w.browser.NameExists(name, 0) {
			return name
		}
	}
}

func (w *Workspace) onBrowseSelect(item browseItem) {
	if item.builtin {
		if bi, ok := w.browser.BuiltinByID(item.id); ok {
			w.openBuiltin(bi)
		}
		return
	}
	w.openCustom(item.id)
}

// openCustom loads a user palette (from memory if modified, else the DB).
func (w *Workspace) openCustom(id int64) {
	if w.current != nil && w.current.id == id {
		return
	}
	w.captureCurrent()
	d := w.docs[id]
	if d == nil {
		entry, err := w.db.Get(id)
		if err != nil {
			dialog.ShowError(err, w.win)
			return
		}
		d = &document{id: id, name: entry.Name, desc: entry.Description, saved: true}
		switch entry.Type {
		case "sine":
			d.kind = docSine
			if r, e := w.db.LoadSine(id); e == nil {
				d.sine = r.Palette
			}
		case "discrete":
			d.kind = docDiscrete
			if r, e := w.db.LoadDiscrete(id); e == nil {
				d.stops = r.Stops
			}
		}
	}
	w.loadDoc(d)
}

// openBuiltin shows a read-only built-in palette in the preview.
func (w *Workspace) openBuiltin(bi builtins.Builtin) {
	w.captureCurrent()
	w.current = nil
	w.readOnly = true
	w.builtinName = bi.Name
	switch bi.Kind {
	case "discrete":
		w.builtinKind = docDiscrete
		w.builtinStops = stopsFromColors(bi.Colors())
		w.discretePreview.SetStops(w.builtinStops)
		w.showEditor(docDiscrete)
	case "sine":
		w.builtinKind = docSine
		if sp, ok := bi.SinePalette(); ok {
			w.builtinSine = sp
			w.sinePreview.SetPalette(sp)
		}
		w.showEditor(docSine)
	}
	w.updateHeader()
}

// makeCopy creates an unsaved copy of a palette (user or built-in) and opens it.
func (w *Workspace) makeCopy(item browseItem) {
	w.captureCurrent()
	var (
		kind  docKind
		sine  gaul.SinePalette
		stops []db.ColorStop
	)
	switch {
	case item.builtin:
		bi, ok := w.browser.BuiltinByID(item.id)
		if !ok {
			return
		}
		if bi.Kind == "sine" {
			kind = docSine
			sine, _ = bi.SinePalette()
		} else {
			kind = docDiscrete
			stops = stopsFromColors(bi.Colors())
		}
	case w.docs[item.id] != nil:
		d := w.docs[item.id]
		kind, sine = d.kind, d.sine
		stops = append([]db.ColorStop(nil), d.stops...)
	default:
		entry, err := w.db.Get(item.id)
		if err != nil {
			dialog.ShowError(err, w.win)
			return
		}
		if entry.Type == "sine" {
			kind = docSine
			if r, e := w.db.LoadSine(item.id); e == nil {
				sine = r.Palette
			}
		} else {
			kind = docDiscrete
			if r, e := w.db.LoadDiscrete(item.id); e == nil {
				stops = r.Stops
			}
		}
	}
	id := w.newSeq
	w.newSeq--
	d := &document{id: id, kind: kind, name: w.copyName(item.name), dirty: true, sine: sine, stops: stops}
	w.docs[id] = d
	w.loadDoc(d)
	w.refreshBrowser()
	w.browser.SelectByID(id)
}

func (w *Workspace) copyName(base string) string {
	if n := base + " (Copy)"; !w.browser.NameExists(n, 0) {
		return n
	}
	for i := 2; ; i++ {
		n := fmt.Sprintf("%s (Copy %d)", base, i)
		if !w.browser.NameExists(n, 0) {
			return n
		}
	}
}

// stopsFromColors turns a color LUT into evenly-spaced color stops.
func stopsFromColors(cols []color.Color) []db.ColorStop {
	n := len(cols)
	stops := make([]db.ColorStop, n)
	for i, c := range cols {
		pos := 0.0
		if n > 1 {
			pos = float64(i) / float64(n-1)
		}
		stops[i] = db.ColorStop{Color: c, Pos: pos}
	}
	return stops
}

// --- Context menu / rename / delete ---

func (w *Workspace) showContextMenu(item browseItem, ev *fyne.PointEvent) {
	items := []*fyne.MenuItem{fyne.NewMenuItem("Make a Copy", func() { w.makeCopy(item) })}
	if !item.builtin {
		items = append(items,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Rename", func() { w.renamePalette(item) }),
			fyne.NewMenuItem("Delete", func() { w.deletePalette(item) }),
		)
	}
	widget.NewPopUpMenu(fyne.NewMenu("", items...), w.win.Canvas()).ShowAtPosition(ev.AbsolutePosition)
}

func (w *Workspace) renamePalette(item browseItem) {
	w.promptName("Rename palette", item.name, func(name string) {
		if name == item.name {
			return
		}
		if w.browser.NameExists(name, item.id) {
			dialog.ShowInformation("Name in use",
				fmt.Sprintf("A palette named %q already exists.", name), w.win)
			return
		}
		if !item.unsaved {
			desc := ""
			if e, err := w.db.Get(item.id); err == nil {
				desc = e.Description
			}
			if err := w.db.Rename(item.id, name, desc); err != nil {
				dialog.ShowError(err, w.win)
				return
			}
		}
		if d := w.docs[item.id]; d != nil {
			d.name = name
		}
		if w.current != nil && w.current.id == item.id {
			w.current.name = name
			w.updateHeader()
		}
		w.refreshBrowser()
		w.browser.SelectByID(item.id)
	})
}

func (w *Workspace) deletePalette(item browseItem) {
	if item.builtin {
		return
	}
	dialog.ShowConfirm("Delete palette", fmt.Sprintf("Delete %q?", item.name), func(ok bool) {
		if !ok {
			return
		}
		if !item.unsaved {
			if err := w.db.Delete(item.id); err != nil {
				dialog.ShowError(err, w.win)
				return
			}
			w.browser.ClearThumbCache()
		}
		delete(w.docs, item.id)
		if w.current != nil && w.current.id == item.id {
			w.current = nil
			w.readOnly = false
			w.showEditor(docNone)
			w.updateHeader()
		}
		w.refreshBrowser()
	}, w.win)
}

func (w *Workspace) itemForCurrent() browseItem {
	d := w.current
	return browseItem{id: d.id, name: d.name, kind: kindString(d.kind), unsaved: !d.saved, dirty: d.dirty}
}

// refreshBrowser rebuilds the Custom section: unsaved documents first, then saved
// DB palettes, with dirty markers.
func (w *Workspace) refreshBrowser() {
	entries, err := w.db.ListAll()
	if err != nil {
		dialog.ShowError(err, w.win)
		return
	}
	var unsaved []*document
	for _, d := range w.docs {
		if !d.saved {
			unsaved = append(unsaved, d)
		}
	}
	sort.Slice(unsaved, func(i, j int) bool { return unsaved[i].id > unsaved[j].id })

	items := make([]browseItem, 0, len(unsaved)+len(entries))
	for _, d := range unsaved {
		items = append(items, browseItem{id: d.id, name: d.name, kind: kindString(d.kind), unsaved: true, dirty: true})
	}
	for _, e := range entries {
		dirty := false
		if d, ok := w.docs[e.ID]; ok && d.dirty {
			dirty = true
		}
		items = append(items, browseItem{id: e.ID, name: e.Name, kind: e.Type, dirty: dirty})
	}
	w.browser.SetCustom(items)
}

// --- Save ---

func (w *Workspace) save() {
	if w.current == nil {
		return // welcome or a read-only built-in: nothing to save
	}
	w.captureCurrent()
	if w.current.saved {
		w.persist(w.current)
		return
	}
	// First save: prompt for a name.
	d := w.current
	w.promptNameDesc(d.name, d.desc, func(name, desc string) {
		if w.browser.NameExists(name, d.id) {
			dialog.ShowInformation("Name in use",
				fmt.Sprintf("A palette named %q already exists. Please choose a different name.", name), w.win)
			return
		}
		d.name, d.desc = name, desc
		w.persist(d)
	})
}

// persist writes a document to the DB (insert if new, update if existing) and
// marks it clean.
func (w *Workspace) persist(d *document) {
	var err error
	if !d.saved {
		var id int64
		switch d.kind {
		case docSine:
			id, err = w.db.SaveSine(d.name, d.desc, d.sine)
		case docDiscrete:
			id, err = w.db.SaveDiscrete(d.name, d.desc, d.stops)
		}
		if err == nil {
			delete(w.docs, d.id)
			d.id, d.saved = id, true
		}
	} else {
		switch d.kind {
		case docSine:
			err = w.db.UpdateSine(d.id, d.sine)
		case docDiscrete:
			err = w.db.UpdateDiscrete(d.id, d.stops)
		}
		if err == nil {
			delete(w.docs, d.id)
		}
	}
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not save %q: %w", d.name, err), w.win)
		return
	}
	d.dirty = false
	w.browser.ClearThumbCache()
	w.refreshBrowser()
	w.browser.SelectByID(d.id)
	w.updateHeader()
}

// saveAllDirty persists every new/modified document under its current name
// (used by the close-time "Save & Close").
func (w *Workspace) saveAllDirty() {
	w.captureCurrent()
	var list []*document
	for _, d := range w.docs {
		list = append(list, d)
	}
	for _, d := range list {
		var err error
		if !d.saved {
			var id int64
			switch d.kind {
			case docSine:
				id, err = w.db.SaveSine(d.name, d.desc, d.sine)
			case docDiscrete:
				id, err = w.db.SaveDiscrete(d.name, d.desc, d.stops)
			}
			if err == nil {
				delete(w.docs, d.id)
				d.id, d.saved, d.dirty = id, true, false
			}
		} else {
			switch d.kind {
			case docSine:
				err = w.db.UpdateSine(d.id, d.sine)
			case docDiscrete:
				err = w.db.UpdateDiscrete(d.id, d.stops)
			}
			if err == nil {
				delete(w.docs, d.id)
				d.dirty = false
			}
		}
	}
}

// onCloseRequested intercepts the window close: if there are unsaved palettes it
// offers to save them, close anyway, or cancel.
func (w *Workspace) onCloseRequested() {
	w.captureCurrent()
	if len(w.docs) == 0 {
		w.win.Close()
		return
	}
	msg := widget.NewLabel(fmt.Sprintf("You have %d palette(s) with unsaved changes.", len(w.docs)))
	var d dialog.Dialog
	saveBtn := widget.NewButton("Save & Close", func() {
		w.saveAllDirty()
		d.Hide()
		w.win.Close()
	})
	saveBtn.Importance = widget.HighImportance
	discardBtn := widget.NewButton("Close anyway", func() {
		d.Hide()
		w.win.Close()
	})
	content := container.NewVBox(msg, container.NewGridWithColumns(2, discardBtn, saveBtn))
	d = dialog.NewCustom("Unsaved palettes", "Cancel", content, w.win)
	d.Show()
}

// promptName shows a single-field name dialog.
func (w *Workspace) promptName(title, initial string, onOK func(string)) {
	ent := widget.NewEntry()
	ent.SetText(initial)
	d := dialog.NewForm(title, "OK", "Cancel", []*widget.FormItem{widget.NewFormItem("Name", ent)}, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(ent.Text)
		if name == "" {
			dialog.ShowInformation("Name required", "Please enter a name.", w.win)
			return
		}
		onOK(name)
	}, w.win)
	d.Resize(fyne.NewSize(360, 140))
	d.Show()
}

// promptNameDesc shows a name+description form dialog.
func (w *Workspace) promptNameDesc(initName, initDesc string, onOK func(name, desc string)) {
	nameEnt := widget.NewEntry()
	nameEnt.SetPlaceHolder("palette name")
	nameEnt.SetText(initName)
	descEnt := widget.NewEntry()
	descEnt.SetPlaceHolder("description (optional)")
	descEnt.SetText(initDesc)

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEnt),
		widget.NewFormItem("Description", descEnt),
	}
	d := dialog.NewForm("Save palette", "Save", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSpace(nameEnt.Text)
		if name == "" {
			dialog.ShowInformation("Name required", "Please enter a palette name.", w.win)
			return
		}
		onOK(name, strings.TrimSpace(descEnt.Text))
	}, w.win)
	d.Resize(fyne.NewSize(420, 200))
	d.Show()
}

// renameCurrent / deleteCurrent operate on the open palette (Edit menu).

func (w *Workspace) renameCurrent() {
	if w.current == nil {
		dialog.ShowInformation("Rename", "Open a palette to rename it.", w.win)
		return
	}
	w.renamePalette(w.itemForCurrent())
}

func (w *Workspace) deleteCurrent() {
	if w.current == nil {
		dialog.ShowInformation("Delete", "Open a palette to delete it.", w.win)
		return
	}
	w.deletePalette(w.itemForCurrent())
}

// --- Export ---

func (w *Workspace) currentPalette() (gaul.Palette, string, bool) {
	if w.current != nil {
		w.captureCurrent()
		switch w.current.kind {
		case docSine:
			sp := w.current.sine
			return &sp, w.current.name, true
		case docDiscrete:
			g := gradientFromStops(w.current.stops)
			return &g, w.current.name, true
		}
	}
	if w.readOnly {
		if w.builtinKind == docSine {
			sp := w.builtinSine
			return &sp, w.builtinName, true
		}
		g := gradientFromStops(w.builtinStops)
		return &g, w.builtinName, true
	}
	return nil, "", false
}

// gradientFromStops builds a position-aware gradient from color stops.
func gradientFromStops(stops []db.ColorStop) gaul.Gradient {
	cols := make([]color.Color, len(stops))
	pos := make([]float64, len(stops))
	for i, s := range stops {
		cols[i] = s.Color
		pos[i] = s.Pos
	}
	return gaul.NewGradientFromColorStops(cols, pos)
}

func (w *Workspace) exportCurrent(format string) {
	p, name, ok := w.currentPalette()
	if !ok {
		dialog.ShowInformation("Export", "Open or create a palette first.", w.win)
		return
	}
	nEntry := widget.NewEntry()
	nEntry.SetText("16")
	items := []*widget.FormItem{widget.NewFormItem("Number of colors", nEntry)}
	dialog.ShowForm("Export "+format, "Choose file…", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		n, err := strconv.Atoi(strings.TrimSpace(nEntry.Text))
		if err != nil || n < 1 {
			dialog.ShowInformation("Export", "Enter a positive number of colors.", w.win)
			return
		}
		colors := export.Sample(p, n)
		fd := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w.win)
				return
			}
			if wc == nil {
				return
			}
			defer wc.Close()
			var werr error
			switch format {
			case "GPL":
				werr = export.WriteGPL(wc, name, colors)
			case "CSV":
				werr = export.WriteCSV(wc, colors)
			}
			if werr != nil {
				dialog.ShowError(werr, w.win)
			}
		}, w.win)
		fd.SetFileName(exportFilename(name, format))
		fd.Show()
	}, w.win)
}

func exportFilename(name, format string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	ext := ".csv"
	if format == "GPL" {
		ext = ".gpl"
	}
	return safe + ext
}

// --- Menu ---

func (w *Workspace) buildMenu() *fyne.MainMenu {
	newSine := fyne.NewMenuItem("New Sine", w.newSine)
	newDiscrete := fyne.NewMenuItem("New Discrete", w.newDiscrete)

	saveItem := fyne.NewMenuItem("Save", w.save)
	saveItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault}

	exportGPL := fyne.NewMenuItem("GPL…", func() { w.exportCurrent("GPL") })
	exportCSV := fyne.NewMenuItem("CSV…", func() { w.exportCurrent("CSV") })
	exportItem := fyne.NewMenuItem("Export", nil)
	exportItem.ChildMenu = fyne.NewMenu("", exportGPL, exportCSV)

	fileMenu := fyne.NewMenu("File",
		newSine, newDiscrete,
		fyne.NewMenuItemSeparator(),
		saveItem,
		fyne.NewMenuItemSeparator(),
		exportItem,
	)

	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Rename…", w.renameCurrent),
		fyne.NewMenuItem("Delete", w.deleteCurrent),
	)

	viewMenu := fyne.NewMenu("View", w.themeMenuItem())

	about := fyne.NewMenuItem("About", w.showAbout)
	helpMenu := fyne.NewMenu("Help", about)

	return fyne.NewMainMenu(fileMenu, editMenu, viewMenu, helpMenu)
}

func (w *Workspace) themeMenuItem() *fyne.MenuItem {
	mk := func(choice string) *fyne.MenuItem {
		it := fyne.NewMenuItem(choice, func() { w.setTheme(choice) })
		it.Checked = w.themeChoice == choice
		return it
	}
	item := fyne.NewMenuItem("Theme", nil)
	item.ChildMenu = fyne.NewMenu("", mk(themeLight), mk(themeDark), mk(themeSystem))
	return item
}

func (w *Workspace) setTheme(choice string) {
	w.themeChoice = choice
	applyThemeChoice(w.app, choice)
	w.app.Preferences().SetString(themePrefKey, choice)
	// Rebuild menu so the checkmarks reflect the new choice.
	w.win.SetMainMenu(w.buildMenu())
}

func (w *Workspace) showAbout() {
	logo := canvas.NewImageFromResource(resources.AppIcon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(96, 96))

	title := widget.NewLabelWithStyle("PaletteDB "+appVersion, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabelWithStyle("A tool for creating, browsing, and\nexporting color palettes.", fyne.TextAlignCenter, fyne.TextStyle{})
	link := widget.NewLabelWithStyle("github.com/aldernero/palettedb", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	content := container.NewVBox(
		container.NewCenter(logo),
		title,
		desc,
		link,
	)
	dialog.NewCustom("About PaletteDB", "Close", content, w.win).Show()
}
