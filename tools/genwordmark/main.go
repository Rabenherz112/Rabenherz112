// Command genwordmark re-encodes the header artwork into the two PNGs the
// header card embeds, one per palette.
//
// This exists so the generator never re-encodes an image at build time.
// image/png's compressor makes no promise that the same pixels produce the same
// bytes on a different Go release, and the cards are checked against the
// committed ones byte for byte, so encoding on every run made a card change the
// day the toolchain moved with nothing else having changed at all. Committing
// the output takes the compressor out of the loop entirely.
//
// Run it from the repository root after changing assets/HeaderTransparent.png
// or anything in internal/wordmark, and commit what it writes:
//
//	go run ./tools/genwordmark
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Rabenherz112/Rabenherz112/internal/wordmark"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("genwordmark: ")

	const (
		src      = "assets/HeaderTransparent.png"
		darkOut  = "assets/wordmark-dark.png"
		lightOut = "assets/wordmark-light.png"
	)

	for _, v := range []struct {
		out string
		// lighten darkens the near-white greys in the artwork, which are drawn
		// for a dark background and vanish on white.
		lighten bool
	}{
		{darkOut, false},
		{lightOut, true},
	} {
		got, err := wordmark.Prepare(src, v.lighten)
		if err != nil {
			log.Fatalf("prepare %s: %v", v.out, err)
		}
		if err := os.MkdirAll(filepath.Dir(v.out), 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(v.out, got.PNG, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s  %dx%d  %d bytes\n", v.out, got.Width, got.Height, len(got.PNG))
	}
}
