package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/builtins"
	"github.com/aldernero/palettedb/internal/db"
	"github.com/aldernero/palettedb/ui/resources"
)

// builtinIDBase offsets synthetic built-in IDs so they are negative and never
// collide with the -1 "no selection" sentinel: built-in i has ID -(i + 2).
const builtinIDBase = 2

// maxSectionRows caps how tall a browse section grows before its list scrolls.
const maxSectionRows = 12

// browseItem is one row in a browse section (a user palette or a built-in).
type browseItem struct {
	id      int64
	name    string
	kind    string // "sine" / "discrete"
	builtin bool
	unsaved bool // new, not yet in the DB → construction placeholder
	dirty   bool // has unsaved changes → "*"
}

// listHeight is a mutable layout that pins its content to a fixed height and the
// full container width (used to size each collapsible section to its contents).
type listHeight struct{ h float32 }

func (l *listHeight) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, l.h) }
func (l *listHeight) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

// Browser is the left-hand sidebar: two collapsible sections (Custom, Built-in).
// The Custom items are supplied by the workspace (which owns the document model);
// the Built-in items are loaded here. Rows support a right-click context menu.
type Browser struct {
	widget.BaseWidget
	database *db.DB
	window   fyne.Window
	builtins []builtins.Builtin

	customItems  []browseItem
	builtinItems []browseItem
	customList   *widget.List
	builtinList  *widget.List
	customWrap   *fyne.Container
	builtinWrap  *fyne.Container
	customH      *listHeight
	builtinH     *listHeight
	accordion    *widget.Accordion
	rowH         float32

	selectedID int64
	cache      map[int64]gaul.Palette

	// OnSelect is invoked when the user selects a palette.
	OnSelect func(browseItem)
	// OnContext is invoked on a right-click, to show a context menu.
	OnContext func(browseItem, *fyne.PointEvent)
}

func NewBrowser(database *db.DB, window fyne.Window) *Browser {
	b := &Browser{
		database:   database,
		window:     window,
		selectedID: -1,
		cache:      make(map[int64]gaul.Palette),
		builtins:   builtins.All(),
		customH:    &listHeight{},
		builtinH:   &listHeight{},
	}
	b.ExtendBaseWidget(b)

	b.customList = b.newSectionList(func() []browseItem { return b.customItems }, func() *widget.List { return b.builtinList })
	b.builtinList = b.newSectionList(func() []browseItem { return b.builtinItems }, func() *widget.List { return b.customList })

	b.rowH = newPaletteRow().MinSize().Height + 1
	b.customWrap = container.New(b.customH, b.customList)
	b.builtinWrap = container.New(b.builtinH, b.builtinList)

	b.refreshBuiltins()
	return b
}

func (b *Browser) newSectionList(itemsFn func() []browseItem, otherFn func() *widget.List) *widget.List {
	l := widget.NewList(
		func() int { return len(itemsFn()) },
		func() fyne.CanvasObject {
			r := newPaletteRow()
			r.onSecondary = func(item browseItem, ev *fyne.PointEvent) {
				if b.OnContext != nil {
					b.OnContext(item, ev)
				}
			}
			return r
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			items := itemsFn()
			if id < 0 || id >= len(items) {
				return
			}
			item := items[id]
			obj.(*paletteRow).update(item, b.paletteFor(item))
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		items := itemsFn()
		if id < 0 || id >= len(items) {
			return
		}
		item := items[id]
		b.selectedID = item.id
		if o := otherFn(); o != nil {
			o.UnselectAll()
		}
		if b.OnSelect != nil {
			b.OnSelect(item)
		}
	}
	return l
}

// SetCustom updates the Custom section's items (called by the workspace).
func (b *Browser) SetCustom(items []browseItem) {
	b.customItems = items
	if b.customList != nil {
		b.customList.Refresh()
		b.resizeSections()
	}
}

// refreshBuiltins (re)builds the Built-in section from the registry.
func (b *Browser) refreshBuiltins() {
	items := make([]browseItem, len(b.builtins))
	for i, bi := range b.builtins {
		items[i] = browseItem{
			id:      int64(-(i + builtinIDBase)),
			name:    bi.Name,
			kind:    bi.Kind,
			builtin: true,
		}
	}
	b.builtinItems = items
	if b.builtinList != nil {
		b.builtinList.Refresh()
		b.resizeSections()
	}
}

// ClearThumbCache drops cached thumbnails (call after content changes).
func (b *Browser) ClearThumbCache() { b.cache = make(map[int64]gaul.Palette) }

// NameExists reports whether a palette (user or built-in) already uses the given
// name, ignoring the item with id `exclude`. Case-insensitive so a saved copy
// can't shadow a built-in.
func (b *Browser) NameExists(name string, exclude int64) bool {
	for _, it := range b.customItems {
		if it.id != exclude && strings.EqualFold(it.name, name) {
			return true
		}
	}
	for _, it := range b.builtinItems {
		if it.id != exclude && strings.EqualFold(it.name, name) {
			return true
		}
	}
	return false
}

// BuiltinByID returns the built-in for a synthetic entry ID (or false).
func (b *Browser) BuiltinByID(id int64) (builtins.Builtin, bool) {
	i := int(-id - builtinIDBase)
	if i < 0 || i >= len(b.builtins) {
		return builtins.Builtin{}, false
	}
	return b.builtins[i], true
}

// paletteFor returns (loading and caching) the thumbnail palette for an item, or
// nil for unsaved items (which use the construction placeholder).
func (b *Browser) paletteFor(item browseItem) gaul.Palette {
	if item.unsaved {
		return nil
	}
	if p, ok := b.cache[item.id]; ok {
		return p
	}
	var p gaul.Palette
	if item.builtin {
		if bi, ok := b.BuiltinByID(item.id); ok {
			p = bi.Palette()
		}
	} else {
		switch item.kind {
		case "sine":
			if r, err := b.database.LoadSine(item.id); err == nil {
				sp := r.Palette
				p = &sp
			}
		case "discrete":
			if r, err := b.database.LoadDiscrete(item.id); err == nil {
				g := r.Gradient()
				p = &g
			}
		}
	}
	b.cache[item.id] = p
	return p
}

func (b *Browser) resizeSections() {
	b.customH.h = b.rowH * float32(clampInt(len(b.customItems), 1, maxSectionRows))
	b.builtinH.h = b.rowH * float32(clampInt(len(b.builtinItems), 1, maxSectionRows))
	if b.customWrap != nil {
		b.customWrap.Refresh()
		b.builtinWrap.Refresh()
	}
	if b.accordion != nil {
		b.accordion.Refresh()
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// UnselectAll clears the selection in both sections.
func (b *Browser) UnselectAll() {
	b.selectedID = -1
	b.customList.UnselectAll()
	b.builtinList.UnselectAll()
}

// SelectByID selects the row for the given id, if present.
func (b *Browser) SelectByID(id int64) {
	for i, it := range b.customItems {
		if it.id == id {
			b.selectedID = id
			b.builtinList.UnselectAll()
			b.customList.Select(i)
			return
		}
	}
	for i, it := range b.builtinItems {
		if it.id == id {
			b.selectedID = id
			b.customList.UnselectAll()
			b.builtinList.Select(i)
			return
		}
	}
}

func (b *Browser) CreateRenderer() fyne.WidgetRenderer {
	b.accordion = widget.NewAccordion(
		widget.NewAccordionItem("Custom", b.customWrap),
		widget.NewAccordionItem("Built-in", b.builtinWrap),
	)
	b.accordion.MultiOpen = true
	b.accordion.OpenAll()
	b.refreshBuiltins()
	content := container.NewScroll(b.accordion)
	return widget.NewSimpleRenderer(content)
}

// --- paletteRow: a right-clickable list row ---

type paletteRow struct {
	widget.BaseWidget
	thumb       *PaletteThumbnail
	constr      *canvas.Image
	nameLbl     *widget.Label
	typeImg     *canvas.Image
	item        browseItem
	onSecondary func(browseItem, *fyne.PointEvent)
}

func newPaletteRow() *paletteRow {
	r := &paletteRow{
		thumb:   NewPaletteThumbnail(nil),
		constr:  canvas.NewImageFromResource(resources.ConstructionIcon),
		nameLbl: widget.NewLabel(""),
		typeImg: canvas.NewImageFromResource(resources.DiscreteIcon),
	}
	r.constr.FillMode = canvas.ImageFillContain
	r.constr.SetMinSize(thumbnailSize)
	r.constr.Hide()
	r.typeImg.FillMode = canvas.ImageFillContain
	r.typeImg.SetMinSize(fyne.NewSize(16, 16))
	r.ExtendBaseWidget(r)
	return r
}

func (r *paletteRow) update(item browseItem, pal gaul.Palette) {
	r.item = item
	name := item.name
	if item.dirty {
		name += " *"
	}
	r.nameLbl.SetText(name)
	if item.kind == "sine" {
		r.typeImg.Resource = resources.SineIcon
	} else {
		r.typeImg.Resource = resources.DiscreteIcon
	}
	r.typeImg.Refresh()
	if item.unsaved {
		r.thumb.Hide()
		r.constr.Show()
	} else {
		r.constr.Hide()
		r.thumb.Show()
		r.thumb.SetPalette(pal)
	}
}

func (r *paletteRow) CreateRenderer() fyne.WidgetRenderer {
	left := container.NewHBox(container.NewStack(r.thumb, r.constr), r.typeImg)
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, left, nil, r.nameLbl))
}

func (r *paletteRow) TappedSecondary(ev *fyne.PointEvent) {
	if r.onSecondary != nil {
		r.onSecondary(r.item, ev)
	}
}

var _ fyne.SecondaryTappable = (*paletteRow)(nil)
