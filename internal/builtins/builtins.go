// Package builtins provides the read-only palettes that ship with palettedb —
// well-known data-visualization colormaps. Each colormap is a listed colormap (a
// sequence of 256 RGB samples) and maps directly onto a discrete palette.
//
// The colormap data and their licenses are third-party; see THIRD_PARTY_NOTICES.md
// and the LICENSE-*.txt files in this directory for attribution.
package builtins

import (
	"bufio"
	"bytes"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"embed"

	"github.com/aldernero/gaul"
)

//go:embed data/*.txt
var dataFS embed.FS

// License identifiers (SPDX where applicable).
const (
	LicenseCC0     = "CC0-1.0"
	LicenseApache  = "Apache-2.0"
	LicenseBSD3    = "BSD-3-Clause"
	LicenseProject = "Project-owned (see repository LICENSE)"
)

// Builtin is a palette that ships with the app. Built-ins are read-only.
type Builtin struct {
	Name        string
	Description string
	Kind        string // "discrete" or "sine"
	License     string
	Author      string
	Source      string

	colors []color.Color     // discrete colormaps
	sine   *gaul.SinePalette // sine palettes (nil for discrete)
}

// Colors returns the discrete colormap's samples (nil for sine built-ins).
func (b Builtin) Colors() []color.Color { return b.colors }

// Palette returns the built-in as a gaul.Palette for previewing/sampling.
func (b Builtin) Palette() gaul.Palette {
	if b.Kind == "sine" && b.sine != nil {
		return b.sine
	}
	g := gaul.NewGradientFromColorStops(b.colors, nil)
	return &g
}

// Gradient returns a discrete built-in as a gaul.Gradient (empty for sine).
func (b Builtin) Gradient() gaul.Gradient {
	return gaul.NewGradientFromColorStops(b.colors, nil)
}

// SinePalette returns a sine built-in's parameters (ok=false for discrete).
func (b Builtin) SinePalette() (gaul.SinePalette, bool) {
	if b.Kind == "sine" && b.sine != nil {
		return *b.sine, true
	}
	return gaul.SinePalette{}, false
}

// ByName returns the built-in with the given name (case-insensitive), or false.
func ByName(name string) (Builtin, bool) {
	for _, b := range registry {
		if strings.EqualFold(b.Name, name) {
			return b, true
		}
	}
	return Builtin{}, false
}

// discreteMeta describes a discrete colormap and its data file (data/<name>.txt).
type discreteMeta struct {
	name, desc, license, author, source string
}

var discreteMetas = []discreteMeta{
	{"viridis", "Perceptually-uniform sequential (matplotlib default)", LicenseCC0,
		"Stéfan van der Walt, Nathaniel J. Smith, Eric Firing", "https://github.com/BIDS/colormap"},
	{"plasma", "Perceptually-uniform sequential", LicenseCC0,
		"Stéfan van der Walt, Nathaniel J. Smith", "https://github.com/BIDS/colormap"},
	{"inferno", "Perceptually-uniform sequential", LicenseCC0,
		"Stéfan van der Walt, Nathaniel J. Smith", "https://github.com/BIDS/colormap"},
	{"magma", "Perceptually-uniform sequential", LicenseCC0,
		"Stéfan van der Walt, Nathaniel J. Smith", "https://github.com/BIDS/colormap"},
	{"cividis", "Perceptually-uniform, optimized for color-vision deficiency", LicenseCC0,
		"Jamie R. Nuñez, Christopher R. Anderton, Ryan S. Renslow", "https://doi.org/10.1371/journal.pone.0199239"},
	{"turbo", "Improved rainbow colormap", LicenseApache,
		"Google LLC (Anton Mikhailov)", "https://gist.github.com/mikhailov-work/ee72ba4191942acecc03fe6da94fc73f"},
	{"mako", "Perceptually-uniform sequential (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
	{"rocket", "Perceptually-uniform sequential (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
	{"flare", "Sequential with restricted luminance (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
	{"crest", "Sequential with restricted luminance (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
	{"vlag", "Diverging blue–red (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
	{"icefire", "Diverging (seaborn)", LicenseBSD3,
		"Michael Waskom (seaborn)", "https://github.com/mwaskom/seaborn"},
}

var registry []Builtin

func init() {
	for _, m := range discreteMetas {
		cols, err := parseLUT(m.name)
		if err != nil {
			panic(err) // embedded data is compiled in; a failure is a build-time bug
		}
		registry = append(registry, Builtin{
			Name:        m.name,
			Description: m.desc,
			Kind:        "discrete",
			License:     m.license,
			Author:      m.author,
			Source:      m.source,
			colors:      cols,
		})
	}
	registry = append(registry, sineBuiltins()...)
	sort.SliceStable(registry, func(i, j int) bool { return registry[i].Name < registry[j].Name })
}

// sineBuiltins returns the project-owned sine palettes that ship with the app.
func sineBuiltins() []Builtin {
	sine := func(name, desc string, c, d gaul.Vec3) Builtin {
		sp := gaul.NewSinePalette(c, d) // A = B = {0.5,0.5,0.5}, C = c, D = d
		return Builtin{
			Name:        name,
			Description: desc,
			Kind:        "sine",
			License:     LicenseProject,
			Author:      "palettedb",
			Source:      "https://github.com/aldernero/palettedb",
			sine:        &sp,
		}
	}
	return []Builtin{
		// The warm-sunset palette shown as the historical "new sine" default.
		sine("warm-sunset", "Warm sunset gradient",
			gaul.Vec3{X: 1, Y: 0.7, Z: 0.3}, gaul.Vec3{X: 0, Y: 0.15, Z: 0.2}),
		sine("rocketpop", "Bright red/pink pop gradient",
			gaul.Vec3{X: 1, Y: 1, Z: 1}, gaul.Vec3{X: 0.120, Y: 0, Z: 0}),
	}
}

// All returns the built-in palettes, sorted by name.
func All() []Builtin { return registry }

// parseLUT reads data/<name>.txt (256 lines of "r g b" floats in [0,1]).
func parseLUT(name string) ([]color.Color, error) {
	b, err := dataFS.ReadFile("data/" + name + ".txt")
	if err != nil {
		return nil, err
	}
	var cols []color.Color
	sc := bufio.NewScanner(bytes.NewReader(b))
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		f := strings.Fields(text)
		if len(f) != 3 {
			return nil, fmt.Errorf("builtins: %s line %d: want 3 fields, got %d", name, line, len(f))
		}
		var v [3]float64
		for i := range f {
			x, err := strconv.ParseFloat(f[i], 64)
			if err != nil {
				return nil, fmt.Errorf("builtins: %s line %d: %w", name, line, err)
			}
			v[i] = clamp01(x)
		}
		cols = append(cols, color.RGBA64{
			R: uint16(v[0]*65535 + 0.5),
			G: uint16(v[1]*65535 + 0.5),
			B: uint16(v[2]*65535 + 0.5),
			A: 65535,
		})
	}
	return cols, sc.Err()
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
