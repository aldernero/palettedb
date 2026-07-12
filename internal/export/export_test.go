package export

import (
	"image/color"
	"strings"
	"testing"
)

func TestWriteGPL(t *testing.T) {
	colors := []color.Color{
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
	}
	var sb strings.Builder
	if err := WriteGPL(&sb, "reds", colors); err != nil {
		t.Fatalf("WriteGPL: %v", err)
	}
	out := sb.String()
	if !strings.HasPrefix(out, "GIMP Palette\n") {
		t.Errorf("missing GIMP header:\n%s", out)
	}
	if !strings.Contains(out, "Name: reds") {
		t.Errorf("missing name:\n%s", out)
	}
	if !strings.Contains(out, "255   0   0\treds-0") {
		t.Errorf("missing red entry:\n%s", out)
	}
}

func TestWriteCSV(t *testing.T) {
	colors := []color.Color{color.RGBA{R: 255, G: 128, B: 0, A: 255}}
	var sb strings.Builder
	if err := WriteCSV(&sb, colors); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "index,r,g,b,hex") {
		t.Errorf("missing header:\n%s", out)
	}
	if !strings.Contains(out, "0,255,128,0,#FF8000") {
		t.Errorf("missing row:\n%s", out)
	}
}
