// Package wordmark prepares the header artwork for embedding into a card.
//
// The source file is a grunge logotype in three parts: "RABENHERZ" in orange,
// "(LP ZOMBIE)" in black, and a "Master of Servers / Ruler of Networks" tagline
// in near-white grey. That grey is drawn for a dark background and all but
// disappears on white, so the light variant folds the light half of the grey
// ramp down into the dark half. Saturated pixels are skipped, which leaves the
// orange exactly as drawn.
//
// The artwork also has to travel inside the card as base64, so it is re-encoded
// as a paletted PNG. It only contains 259 distinct colours, so a 256-entry
// palette holds essentially all of it and stores it in about half the space the
// original truecolour file took, at full resolution. Downscaling would be
// counterproductive: resampling invents thousands of intermediate colours and
// makes the file larger than the original.
//
// That re-encoding happens in tools/genwordmark, not here, and its output is
// committed. It has to: image/png's compressor makes no promise that the same
// pixels give the same bytes on a different Go release, and the generated cards
// are checked against the committed ones byte for byte. Encoding at build time
// meant a card changed the day the toolchain moved, with no change to the
// artwork or the code. Read reads what the tool produced; Prepare is the tool.
package wordmark

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sort"
)

// MIME is the type the embedded data URI is tagged with.
const MIME = "image/png"

// maxPalette is the largest palette a PNG can carry.
const maxPalette = 256

// Prepared is the artwork ready to embed, with the dimensions the card needs in
// order to reserve the right aspect ratio for it.
type Prepared struct {
	PNG           []byte
	Width, Height int
}

// Read loads an already-prepared PNG, the kind tools/genwordmark writes. The
// bytes are embedded in a card exactly as they are on disk.
func Read(path string) (Prepared, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Prepared{}, err
	}
	// DecodeConfig reads the header only, which is all the card needs in order
	// to reserve the right aspect ratio.
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return Prepared{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return Prepared{PNG: b, Width: cfg.Width, Height: cfg.Height}, nil
}

// Prepare reads the source artwork and re-encodes it. When lighten is true, the
// near-white greys are darkened for a light background.
//
// This is called by tools/genwordmark, not by the generator. See the package
// comment for why the result is committed rather than made on every run.
func Prepare(path string, lighten bool) (Prepared, error) {
	f, err := os.Open(path)
	if err != nil {
		return Prepared{}, err
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		return Prepared{}, fmt.Errorf("decode %s: %w", path, err)
	}

	// Work in straight (non-premultiplied) alpha, so a colour can be adjusted
	// without having to divide the alpha back out of it first.
	b := src.Bounds()
	img := image.NewNRGBA(b)
	draw.Draw(img, b, src, b.Min, draw.Src)

	if lighten {
		foldGreys(img)
	}

	// draw.Draw maps any colour left out of the palette to its nearest
	// neighbour. With 259 colours in and 256 slots, that is the three rarest
	// antialiasing steps snapping to a neighbour nobody will find.
	pal := palette(img)
	out := image.NewPaletted(b, pal)
	draw.Draw(out, b, img, b.Min, draw.Src)

	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, out); err != nil {
		return Prepared{}, err
	}
	return Prepared{PNG: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// palette collects the image's colours, most used first, capped at what a PNG
// palette can hold. Fully transparent pixels are normalised to a single entry,
// which is placed first so the large empty margins all share one slot.
//
// The ordering has to be total, not just by frequency. Colours are gathered
// from a map, whose iteration order is random, so ties broken arbitrarily would
// shuffle the palette between runs and change the encoded bytes of an image
// that had not changed at all. The workflow commits what it generates, so that
// would mean a new header card committed on every single run, forever.
func palette(img *image.NRGBA) color.Palette {
	counts := make(map[color.NRGBA]int)
	for i := 0; i < len(img.Pix); i += 4 {
		c := color.NRGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
		if c.A == 0 {
			c = color.NRGBA{}
		}
		counts[c]++
	}

	type entry struct {
		c color.NRGBA
		n int
	}
	all := make([]entry, 0, len(counts))
	for c, n := range counts {
		all = append(all, entry{c, n})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return key(all[i].c) < key(all[j].c)
	})

	pal := color.Palette{color.NRGBA{}}
	for _, e := range all {
		if len(pal) == maxPalette {
			break
		}
		if e.c.A == 0 {
			continue // already in, as the first entry
		}
		pal = append(pal, e.c)
	}
	return pal
}

// key packs a colour into one comparable number, to break frequency ties in a
// stable order.
func key(c color.NRGBA) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}

// greyThreshold is how far apart the RGB channels may be for a pixel to count
// as grey. The orange is far outside it, so it is never touched.
const greyThreshold = 40

// foldGreys darkens grey pixels lighter than mid-grey, folding the top half of
// the ramp onto the bottom half. Pixels at or below mid-grey keep their value,
// so the black lettering and grunge, which already read on white, are left
// alone.
func foldGreys(img *image.NRGBA) {
	const (
		mid      = 128
		compress = 0.7
	)

	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] == 0 {
			continue
		}
		r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]

		hi, lo := r, r
		for _, v := range [2]uint8{g, b} {
			if v > hi {
				hi = v
			}
			if v < lo {
				lo = v
			}
		}
		if int(hi)-int(lo) > greyThreshold {
			continue // coloured: leave the orange alone
		}

		lum := (int(r) + int(g) + int(b)) / 3
		if lum <= mid {
			continue
		}
		v := uint8(float64(mid) - float64(lum-mid)*compress)
		img.Pix[i], img.Pix[i+1], img.Pix[i+2] = v, v, v
	}
}
