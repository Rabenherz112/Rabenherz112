package svgx

import (
	"encoding/xml"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

func TestTextWidth(t *testing.T) {
	// Every glyph advances 0.6em, so three characters at 10px are 18px wide.
	if got, want := TextWidth("abc", 10, 0), 18.0; got != want {
		t.Errorf("TextWidth = %v, want %v", got, want)
	}
	// Letter spacing is added after every glyph, the last one included, which
	// is what browsers do and what TextMiddle compensates for.
	if got, want := TextWidth("abc", 10, 2), 24.0; got != want {
		t.Errorf("TextWidth with spacing = %v, want %v", got, want)
	}
	if got, want := TextWidth("", 10, 2), 0.0; got != want {
		t.Errorf("TextWidth of empty = %v, want %v", got, want)
	}
}

func TestFitChars(t *testing.T) {
	if got, want := FitChars(60, 10, 0), 10; got != want {
		t.Errorf("FitChars = %d, want %d", got, want)
	}
	if got, want := FitChars(0, 10, 0), 0; got != want {
		t.Errorf("FitChars of no width = %d, want %d", got, want)
	}
	if got, want := FitChars(60, 0, 0), 0; got != want {
		t.Errorf("FitChars at zero size = %d, want %d", got, want)
	}
}

func TestWrap(t *testing.T) {
	// A 60px measure at 10px holds ten characters.
	got := Wrap("aaa bbb ccc", 10, 0, 60)
	want := []string{"aaa bbb", "ccc"}
	if len(got) != len(want) {
		t.Fatalf("Wrap = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Wrap line %d = %q, want %q", i, got[i], want[i])
		}
	}

	// A word too long for the measure gets its own line rather than being
	// broken mid-word.
	got = Wrap("aaaaaaaaaaaaaaa bb", 10, 0, 60)
	if len(got) != 2 || got[0] != "aaaaaaaaaaaaaaa" {
		t.Errorf("Wrap of an overlong word = %q", got)
	}

	if got := Wrap("anything", 10, 0, 0); got != nil {
		t.Errorf("Wrap into no space = %q, want nil", got)
	}
}

func TestTruncate(t *testing.T) {
	// Ten characters fit in 60px at 10px, so this is returned untouched.
	if got, want := Truncate("abcdefghij", 10, 0, 60), "abcdefghij"; got != want {
		t.Errorf("Truncate = %q, want %q", got, want)
	}
	// Five characters fit: four kept plus the ellipsis.
	if got, want := Truncate("abcdefghij", 10, 0, 30), "abcd"+Ellipsis; got != want {
		t.Errorf("Truncate = %q, want %q", got, want)
	}
	// The result never overflows the space it was given.
	if w := TextWidth(Truncate("abcdefghij", 10, 0, 30), 10, 0); w > 30 {
		t.Errorf("truncated run is %v wide, want <= 30", w)
	}
	if got, want := Truncate("abc", 10, 0, 0), ""; got != want {
		t.Errorf("Truncate into no space = %q, want %q", got, want)
	}
	// A trailing space before the ellipsis reads as a gap, so it is dropped.
	if got, want := Truncate("ab cdef", 10, 0, 24), "ab"+Ellipsis; got != want {
		t.Errorf("Truncate at a space = %q, want %q", got, want)
	}
}

// closeTo compares two lengths in pixels, allowing for the rounding error that
// comes of accumulating floating point coordinates.
func closeTo(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestBaselines(t *testing.T) {
	// With line-height:normal there is no leading to split, so the baseline
	// sits one ascent below the top.
	if got, want := FirstBaseline(0, 10*NormalLineHeight, 10), 10.2; !closeTo(got, want) {
		t.Errorf("FirstBaseline = %v, want %v", got, want)
	}
	// Extra line height is split evenly above and below, as CSS does.
	if got, want := FirstBaseline(0, 20, 10), 13.6; !closeTo(got, want) {
		t.Errorf("FirstBaseline with leading = %v, want %v", got, want)
	}
	// Optical centring balances the cap height, not the whole line box.
	if got, want := MiddleBaseline(10, 10), 13.65; !closeTo(got, want) {
		t.Errorf("MiddleBaseline = %v, want %v", got, want)
	}
}

// TestMarkScalePrecision guards a bug that made the Windows logo 62% too big.
// Coordinates are rounded to hundredths, which is plenty for a position but
// destroys a scale factor: fitting a 4875-unit viewBox into 30px is 0.00615,
// and rounding that to two decimals gives 0.01.
func TestMarkScalePrecision(t *testing.T) {
	c := NewCard(100, theme.Dark)
	c.Mark(0, 0, 30, 4875, []string{"M0 0h10v10H0z"}, "#000")

	got := c.body.String()
	if !strings.Contains(got, "scale(0.00615385)") {
		t.Errorf("Mark emitted %q, want a scale of 0.00615385", got)
	}
}

// TestRenderIsWellFormed checks that a card parses as XML, including when the
// text came from an API and carries characters that are markup in XML.
func TestRenderIsWellFormed(t *testing.T) {
	c := NewCard(400, theme.Dark)
	c.Text(10, 20, `Fullmetal Alchemist & "Brotherhood" <ova>`, Text{Size: 12, Fill: "#fff"})
	c.Rect(0, 0, 10, 10, "#fff")
	c.Divider(0, 30, 100)
	c.Mark(0, 40, 24, 24, []string{"M0 0h24v24H0z"}, "#fff")
	c.Image(0, 60, 10, 10, "image/png", []byte{1, 2, 3})

	dec := xml.NewDecoder(strings.NewReader(string(c.Render(200))))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("rendered card is not well-formed XML: %v", err)
		}
	}
}

func TestRenderCarriesTheFont(t *testing.T) {
	// Without the embedded face the card falls back to the viewer's monospace
	// and every measurement in this package is wrong.
	got := string(NewCard(100, theme.Dark).Render(50))
	if !strings.Contains(got, "@font-face") || !strings.Contains(got, "data:font/woff2;base64,") {
		t.Error("rendered card does not embed the font")
	}
}

func TestNumTrimsCleanly(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{100, "100"},
		{10.5, "10.5"},
		{10.25, "10.25"},
		{1000, "1000"},
		{-3.5, "-3.5"},
	} {
		if got := num(tc.in); got != tc.want {
			t.Errorf("num(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
