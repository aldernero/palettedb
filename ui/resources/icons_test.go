package resources

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// TestThemedIconsColorize verifies the themed icons are recolored to the
// theme's foreground color. The Material Symbols sources put fill="#1f1f1f" on
// the <svg> root with no fill on the <path>; Fyne's colorizer replaces any
// path fill that isn't "none", so the rendered path must carry the theme color.
func TestThemedIconsColorize(t *testing.T) {
	test.NewApp() // initialize a headless app so theme lookups work
	fg := theme.Color(theme.ColorNameForeground)
	wantHex := hexOf(fg)

	icons := map[string]fyne.Resource{
		"dice":     DiceIcon,
		"link":     LinkIcon,
		"link-off": LinkOffIcon,
		"lock":     LockIcon,
		"unlock":   UnlockIcon,
	}
	for name, res := range icons {
		content := string(res.Content())
		if !strings.Contains(content, "<path") {
			t.Errorf("%s: colorized svg has no <path>: %q", name, content)
		}
		if !strings.Contains(strings.ToLower(content), wantHex) {
			t.Errorf("%s: expected foreground color %s in colorized svg, got:\n%s", name, wantHex, content)
		}
	}
}

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
