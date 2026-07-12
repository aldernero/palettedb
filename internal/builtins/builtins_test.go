package builtins

import (
	"fmt"
	"image/color"
	"testing"
)

func to8(x uint32) uint8 { return uint8((x*255 + 32767) / 65535) }

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", to8(r), to8(g), to8(b))
}

func TestAllLoad(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("no built-ins registered")
	}
	for _, b := range all {
		if b.License == "" || b.Author == "" || b.Source == "" {
			t.Errorf("%s: missing attribution (license=%q author=%q source=%q)", b.Name, b.License, b.Author, b.Source)
		}
		if b.Kind == "discrete" && len(b.Colors()) != 256 {
			t.Errorf("%s: expected 256 colors, got %d", b.Name, len(b.Colors()))
		}
		if b.Palette() == nil {
			t.Errorf("%s: nil palette", b.Name)
		}
	}
}

func TestEndpoints(t *testing.T) {
	byName := map[string]Builtin{}
	for _, b := range All() {
		byName[b.Name] = b
	}
	checks := []struct{ name, first, last string }{
		{"viridis", "#440154", "#FDE725"},
		{"plasma", "#0D0887", "#F0F921"},
		{"cividis", "#00224E", "#FEE838"},
		{"turbo", "#30123B", "#7A0403"},
		{"mako", "#0B0405", "#DEF5E5"},
	}
	for _, c := range checks {
		b, ok := byName[c.name]
		if !ok {
			t.Errorf("missing built-in %q", c.name)
			continue
		}
		cs := b.Colors()
		if got := hexOf(cs[0]); got != c.first {
			t.Errorf("%s first = %s, want %s", c.name, got, c.first)
		}
		if got := hexOf(cs[len(cs)-1]); got != c.last {
			t.Errorf("%s last = %s, want %s", c.name, got, c.last)
		}
	}
}
