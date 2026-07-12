package palettedb_test

import (
	"fmt"
	"log"

	"github.com/aldernero/palettedb"
)

// Example shows fetching palettes by name. By-name lookups search the user's
// saved palettes first, then the built-ins, so "viridis" and "warm-sunset"
// resolve even against an empty database.
func Example() {
	db, err := palettedb.OpenDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// A discrete palette (built-in) as a gaul.Gradient.
	grad, err := db.LoadDiscreteByName("viridis")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		r, g, b, _ := grad.ColorAtStop(i, 8).RGBA()
		fmt.Printf("%d: #%02X%02X%02X\n", i, uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	// A sine palette by name.
	sp, err := db.LoadSineByName("warm-sunset")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sp.ColorAt(0.5))

	// When the type is unknown, PaletteByName returns a gaul.Palette + its kind.
	p, kind, err := db.PaletteByName("turbo")
	if err != nil {
		log.Fatal(err)
	}
	_ = p.ColorAt(0.42)
	fmt.Println(kind)
}
