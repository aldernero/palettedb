package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme choice preference values.
const (
	themeLight  = "Light"
	themeDark   = "Dark"
	themeSystem = "System"
	themePrefKey = "theme"
)

// forcedVariantTheme wraps the default theme but pins the light/dark variant,
// ignoring the OS setting. Used for the explicit Light/Dark choices.
type forcedVariantTheme struct {
	variant fyne.ThemeVariant
}

func (t forcedVariantTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, t.variant)
}

func (t forcedVariantTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

func (t forcedVariantTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (t forcedVariantTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}

// applyThemeChoice sets the app theme for the given choice. "System" follows the
// OS setting (Fyne updates it automatically); Light/Dark force a variant.
func applyThemeChoice(a fyne.App, choice string) {
	switch choice {
	case themeLight:
		a.Settings().SetTheme(forcedVariantTheme{variant: theme.VariantLight})
	case themeDark:
		a.Settings().SetTheme(forcedVariantTheme{variant: theme.VariantDark})
	default:
		a.Settings().SetTheme(theme.DefaultTheme())
	}
}
