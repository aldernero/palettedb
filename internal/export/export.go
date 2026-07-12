// Package export writes palettes to interchange formats (GIMP .gpl and CSV).
package export

import (
	"bufio"
	"fmt"
	"image/color"
	"io"

	"github.com/aldernero/gaul"
)

// Sample returns n colors evenly sampled from a palette using ColorAtStop.
func Sample(p gaul.Palette, n int) []color.Color {
	if n < 1 {
		n = 1
	}
	colors := make([]color.Color, n)
	for i := 0; i < n; i++ {
		colors[i] = p.ColorAtStop(i, n)
	}
	return colors
}

// rgb8 converts a color to 8-bit R,G,B components.
func rgb8(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return to8(r), to8(g), to8(b)
}

// to8 converts a 16-bit color channel (0..65535) to 8 bits, rounding to nearest
// (matching the round(v*255) convention used by matplotlib and most tools).
func to8(x uint32) uint8 { return uint8((x*255 + 32767) / 65535) }

// WriteGPL writes colors in GIMP palette (.gpl) format.
func WriteGPL(w io.Writer, name string, colors []color.Color) error {
	bw := bufio.NewWriter(w)
	if name == "" {
		name = "Untitled"
	}
	fmt.Fprintln(bw, "GIMP Palette")
	fmt.Fprintf(bw, "Name: %s\n", name)
	fmt.Fprintln(bw, "Columns: 0")
	fmt.Fprintln(bw, "#")
	for i, c := range colors {
		r, g, b := rgb8(c)
		fmt.Fprintf(bw, "%3d %3d %3d\t%s-%d\n", r, g, b, name, i)
	}
	return bw.Flush()
}

// WriteCSV writes colors as CSV with header index,r,g,b,hex.
func WriteCSV(w io.Writer, colors []color.Color) error {
	bw := bufio.NewWriter(w)
	fmt.Fprintln(bw, "index,r,g,b,hex")
	for i, c := range colors {
		r, g, b := rgb8(c)
		fmt.Fprintf(bw, "%d,%d,%d,%d,#%02X%02X%02X\n", i, r, g, b, r, g, b)
	}
	return bw.Flush()
}
