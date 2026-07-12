package ui

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/aldernero/gaul"
	"github.com/aldernero/palettedb/internal/db"
)

func newTestWorkspace(t *testing.T) *Workspace {
	t.Helper()
	a := test.NewApp()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	win := a.NewWindow("test")
	ws := &Workspace{app: a, win: win, db: database}
	ws.build()
	return ws
}

func TestNewDocIsUnsavedAndDirty(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.newSine()
	if ws.current == nil {
		t.Fatal("expected a current document")
	}
	if ws.current.saved || !ws.current.dirty {
		t.Errorf("new doc should be unsaved+dirty, got saved=%v dirty=%v", ws.current.saved, ws.current.dirty)
	}
	if ws.current.name != "Untitled #1" {
		t.Errorf("name = %q, want Untitled #1", ws.current.name)
	}
	if len(ws.docs) != 1 {
		t.Errorf("docs = %d, want 1", len(ws.docs))
	}
	// It shows in the Custom section as an unsaved item.
	if len(ws.browser.customItems) != 1 || !ws.browser.customItems[0].unsaved {
		t.Errorf("expected 1 unsaved custom item, got %+v", ws.browser.customItems)
	}
}

func TestSaveMovesToDB(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.newDiscrete()
	ws.current.name = "my-grad" // stand in for the name prompt
	ws.persist(ws.current)
	if !ws.current.saved || ws.current.dirty {
		t.Errorf("after save: saved=%v dirty=%v", ws.current.saved, ws.current.dirty)
	}
	if len(ws.docs) != 0 {
		t.Errorf("docs = %d, want 0 after save", len(ws.docs))
	}
	if _, err := ws.db.GetByName("my-grad"); err != nil {
		t.Errorf("palette not in DB: %v", err)
	}
	// Now shows as a saved (not unsaved) custom item.
	if len(ws.browser.customItems) != 1 || ws.browser.customItems[0].unsaved {
		t.Errorf("expected 1 saved custom item, got %+v", ws.browser.customItems)
	}
}

func TestMakeCopyOfBuiltin(t *testing.T) {
	ws := newTestWorkspace(t)
	var viridis browseItem
	for _, it := range ws.browser.builtinItems {
		if it.name == "viridis" {
			viridis = it
		}
	}
	ws.makeCopy(viridis)
	if ws.current == nil || ws.current.saved {
		t.Fatal("copy should be an unsaved current doc")
	}
	if ws.current.name != "viridis (Copy)" {
		t.Errorf("copy name = %q", ws.current.name)
	}
	if ws.current.kind != docDiscrete || len(ws.current.stops) != 256 {
		t.Errorf("viridis copy should be a 256-stop discrete, got kind=%v stops=%d", ws.current.kind, len(ws.current.stops))
	}
}

func TestEditExistingMarksDirty(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.db.SaveSine("existing", "", gaul.NewSinePalette(
		gaul.Vec3{X: 1, Y: 0.7, Z: 0.3}, gaul.Vec3{X: 0, Y: 0.15, Z: 0.2}))
	ws.refreshBrowser()

	var it browseItem
	for _, x := range ws.browser.customItems {
		if x.name == "existing" {
			it = x
		}
	}
	ws.onBrowseSelect(it)
	if ws.current == nil || ws.current.dirty || !ws.current.saved {
		t.Fatalf("opened existing should be clean+saved, got %+v", ws.current)
	}
	// Simulate an edit.
	ws.onEditorChanged()
	if !ws.current.dirty {
		t.Error("editing should mark the document dirty")
	}
	if _, ok := ws.docs[ws.current.id]; !ok {
		t.Error("a dirty existing document should be tracked in docs")
	}
	found := false
	for _, x := range ws.browser.customItems {
		if x.id == ws.current.id && x.dirty {
			found = true
		}
	}
	if !found {
		t.Error("expected the dirty marker to show in the list")
	}
}

func TestSaveAllDirty(t *testing.T) {
	ws := newTestWorkspace(t)
	ws.newSine()
	ws.newDiscrete() // two unsaved docs
	if len(ws.docs) != 2 {
		t.Fatalf("docs = %d, want 2", len(ws.docs))
	}
	ws.saveAllDirty()
	if len(ws.docs) != 0 {
		t.Errorf("docs = %d, want 0 after saveAllDirty", len(ws.docs))
	}
	all, _ := ws.db.ListAll()
	if len(all) != 2 {
		t.Errorf("DB has %d palettes, want 2", len(all))
	}
}
