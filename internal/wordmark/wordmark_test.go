package wordmark

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

const artwork = "../../assets/HeaderTransparent.png"

// TestPrepareIsDeterministic guards a bug that made every run produce a different
// header card. The palette is built from a map, and Go randomises map
// iteration, so ties in colour frequency were being broken differently each
// time. The bytes changed, the workflow saw a diff, and it would have committed
// a "new" header on every single run.
func TestPrepareIsDeterministic(t *testing.T) {
	for _, lighten := range []bool{false, true} {
		first, err := Prepare(artwork, lighten)
		if err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		for i := range 4 {
			again, err := Prepare(artwork, lighten)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			if !bytes.Equal(first.PNG, again.PNG) {
				t.Fatalf("lighten=%v: run %d produced different bytes (%d vs %d)",
					lighten, i+2, len(first.PNG), len(again.PNG))
			}
		}
	}
}

func TestPrepareKeepsDimensions(t *testing.T) {
	got, err := Prepare(artwork, false)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	// The artwork is re-encoded, not resampled: resizing would invent
	// intermediate colours and make the file larger than the original.
	if got.Width != 1491 || got.Height != 434 {
		t.Errorf("size = %dx%d, want the source's 1491x434", got.Width, got.Height)
	}
	if len(got.PNG) == 0 {
		t.Fatal("no PNG data")
	}
}

// TestPrepareIsSmallerThanTheSource checks the paletted re-encode is actually
// worth doing. The artwork travels inside the card as base64, so its size is
// paid on every profile view.
func TestPrepareIsSmallerThanTheSource(t *testing.T) {
	got, err := Prepare(artwork, false)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	const sourceBytes = 78834
	if len(got.PNG) >= sourceBytes {
		t.Errorf("re-encoded to %d bytes, want well under the source's %d", len(got.PNG), sourceBytes)
	}
}

func TestFoldGreys(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 1))
	set := func(x int, c color.NRGBA) {
		i := x * 4
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}

	orange := color.NRGBA{0xd7, 0x85, 0x42, 0xff} // the wordmark's accent
	set(0, color.NRGBA{0xd2, 0xd2, 0xd2, 0xff})   // the tagline's near-white
	set(1, color.NRGBA{0x00, 0x00, 0x00, 0xff})   // the black lettering
	set(2, color.NRGBA{0x7f, 0x7f, 0x7f, 0xff})   // mid grey, at the fold
	set(3, orange)
	set(4, color.NRGBA{0xff, 0xff, 0xff, 0x00}) // transparent

	foldGreys(img)

	at := func(x int) color.NRGBA {
		i := x * 4
		return color.NRGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
	}

	// The near-white tagline is pulled well below mid grey so it reads on a
	// light background.
	if got := at(0); got.R >= 0x80 {
		t.Errorf("near-white folded to %v, want something dark", got)
	}
	// Black already reads on white and is left alone.
	if got := at(1); got != (color.NRGBA{0, 0, 0, 0xff}) {
		t.Errorf("black became %v, want it untouched", got)
	}
	// Mid grey is the hinge and does not move.
	if got := at(2); got.R > 0x80 {
		t.Errorf("mid grey became %v, want it at or below the fold", got)
	}
	// The orange is saturated, so it is never touched.
	if got := at(3); got != orange {
		t.Errorf("orange became %v, want %v", got, orange)
	}
	if got := at(4); got.A != 0 {
		t.Errorf("transparent pixel gained alpha: %v", got)
	}
}

// TestCommittedWordmarksMatchTheTool checks the PNGs in assets/ still show
// what tools/genwordmark would produce today.
//
// They are committed rather than made on every run, so nothing else would
// notice if the artwork or the folding logic changed and the tool was not
// re-run: the cards would go on embedding a stale image indefinitely.
//
// The comparison is pixel by pixel, deliberately not byte for byte. Comparing
// encoded bytes is what broke the build in the first place: image/png's
// compressor gives different output on a different Go release, so a byte
// comparison here would fail on any runner whose toolchain differs from the
// machine that last ran the tool, which is the exact fragility this whole
// arrangement exists to remove. What matters is that the picture is right.
func TestCommittedWordmarksMatchTheTool(t *testing.T) {
	for _, v := range []struct {
		path    string
		lighten bool
	}{
		{"../../assets/wordmark-dark.png", false},
		{"../../assets/wordmark-light.png", true},
	} {
		want, err := Prepare(artwork, v.lighten)
		if err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		got, err := Read(v.path)
		if err != nil {
			t.Fatalf("Read %s: %v", v.path, err)
		}
		if got.Width != want.Width || got.Height != want.Height {
			t.Errorf("%s is %dx%d, want %dx%d. Run: go run ./tools/genwordmark",
				v.path, got.Width, got.Height, want.Width, want.Height)
			continue
		}
		if diff := pixelDiff(t, got.PNG, want.PNG); diff > 0 {
			t.Errorf("%s differs from the tool's output in %d pixels. "+
				"Run: go run ./tools/genwordmark", v.path, diff)
		}
	}
}

// TestCommittedWordmarksDiffer is a sanity check on the pair. If the light
// variant were accidentally written from the unfolded artwork, the two files
// would be identical and the tagline would vanish on a white page, which is
// the one thing the light variant exists to prevent.
func TestCommittedWordmarksDiffer(t *testing.T) {
	dark, err := Read("../../assets/wordmark-dark.png")
	if err != nil {
		t.Fatal(err)
	}
	light, err := Read("../../assets/wordmark-light.png")
	if err != nil {
		t.Fatal(err)
	}
	if pixelDiff(t, dark.PNG, light.PNG) == 0 {
		t.Error("the two wordmarks are pixel-identical, so the light one was not folded")
	}
}

// pixelDiff decodes both PNGs and counts how many pixels disagree.
func pixelDiff(t *testing.T, a, b []byte) int {
	t.Helper()
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ia.Bounds() != ib.Bounds() {
		t.Fatalf("bounds differ: %v vs %v", ia.Bounds(), ib.Bounds())
	}

	diff := 0
	for y := ia.Bounds().Min.Y; y < ia.Bounds().Max.Y; y++ {
		for x := ia.Bounds().Min.X; x < ia.Bounds().Max.X; x++ {
			if ia.At(x, y) != ib.At(x, y) {
				diff++
			}
		}
	}
	return diff
}
