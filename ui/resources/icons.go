// Package resources holds embedded icon assets for the palettedb UI.
//
// dice.svg, link.svg, link-off.svg, palette.svg, discrete.svg, sine.svg, and
// construction.svg are from Google Material Symbols (https://fonts.google.com/icons),
// © Google LLC, licensed under Apache-2.0. See THIRD_PARTY_NOTICES.md and
// LICENSE-Apache-2.0.txt.
package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed lock.svg
var lockSVG []byte

//go:embed unlock.svg
var unlockSVG []byte

//go:embed dice.svg
var diceSVG []byte

//go:embed link.svg
var linkSVG []byte

//go:embed link-off.svg
var linkOffSVG []byte

//go:embed palette.svg
var paletteSVG []byte

//go:embed discrete.svg
var discreteSVG []byte

//go:embed sine.svg
var sineSVG []byte

//go:embed construction.svg
var constructionSVG []byte

// icon.png is generated from icon.svg (Inkscape) because Fyne's SVG rasterizer
// flattens gradients; the PNG preserves them. Regenerate with:
//
//	inkscape icon.svg --export-type=png --export-filename=icon.png -w 256 -h 256
//
//go:embed icon.png
var iconPNG []byte

// Themed monochrome glyphs. These recolor to the active theme's foreground
// color (Fyne only replaces the SVG fill, so the assets are fill-only).
var (
	LockIcon    = theme.NewThemedResource(fyne.NewStaticResource("lock.svg", lockSVG))
	UnlockIcon  = theme.NewThemedResource(fyne.NewStaticResource("unlock.svg", unlockSVG))
	DiceIcon    = theme.NewThemedResource(fyne.NewStaticResource("dice.svg", diceSVG))
	LinkIcon    = theme.NewThemedResource(fyne.NewStaticResource("link.svg", linkSVG))
	LinkOffIcon = theme.NewThemedResource(fyne.NewStaticResource("link-off.svg", linkOffSVG))
	PaletteIcon = theme.NewThemedResource(fyne.NewStaticResource("palette.svg", paletteSVG))
	// Palette-type glyphs for browse-list rows.
	DiscreteIcon = theme.NewThemedResource(fyne.NewStaticResource("discrete.svg", discreteSVG))
	SineIcon     = theme.NewThemedResource(fyne.NewStaticResource("sine.svg", sineSVG))
	// ConstructionIcon marks an unsaved (under-construction) palette.
	ConstructionIcon = theme.NewThemedResource(fyne.NewStaticResource("construction.svg", constructionSVG))
)

// AppIcon is the multi-color application/window icon (not themed). Also shown
// in the About dialog.
var AppIcon = fyne.NewStaticResource("icon.png", iconPNG)
