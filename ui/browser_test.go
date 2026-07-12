package ui

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/aldernero/palettedb/internal/builtins"
	"github.com/aldernero/palettedb/internal/db"
)

// TestBrowserBuiltins checks the Built-in section is populated from the registry
// and that BuiltinByID / paletteFor resolve built-ins.
func TestBrowserBuiltins(t *testing.T) {
	test.NewApp()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	w := test.NewWindow(nil)
	defer w.Close()
	b := NewBrowser(database, w)

	if len(b.builtinItems) != len(builtins.All()) {
		t.Fatalf("built-in items = %d, want %d", len(b.builtinItems), len(builtins.All()))
	}
	for _, it := range b.builtinItems {
		if !it.builtin {
			t.Errorf("%s not marked builtin", it.name)
		}
		if _, ok := b.BuiltinByID(it.id); !ok {
			t.Errorf("BuiltinByID failed for %s (id=%d)", it.name, it.id)
		}
		if b.paletteFor(it) == nil {
			t.Errorf("nil palette for built-in %s", it.name)
		}
	}
}

func TestNameExists(t *testing.T) {
	test.NewApp()
	database, _ := db.OpenAt(filepath.Join(t.TempDir(), "t.db"))
	defer database.Close()
	b := NewBrowser(database, test.NewWindow(nil))
	// The workspace normally supplies custom items; simulate one here.
	b.SetCustom([]browseItem{{id: 1, name: "mine", kind: "discrete"}})

	// Built-in name is taken (case-insensitively) — prevents shadowing.
	if !b.NameExists("viridis", 0) || !b.NameExists("Viridis", 0) {
		t.Error("expected built-in name 'viridis' to be reported as taken")
	}
	// A custom palette name is taken.
	if !b.NameExists("mine", 0) {
		t.Error("expected user name 'mine' to be taken")
	}
	// A fresh name is free.
	if b.NameExists("brand-new", 0) {
		t.Error("expected 'brand-new' to be free")
	}
	// Excluding the item's own id lets a rename keep its name.
	if b.NameExists("mine", 1) {
		t.Error("renaming to the same name should be allowed (exclude self)")
	}
}
