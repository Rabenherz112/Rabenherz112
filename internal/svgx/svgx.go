// Package svgx draws the cards.
//
// Every card is a standalone SVG that GitHub loads through an <img> tag, which
// means no external stylesheet, no script, no web font request and no HTML
// layout engine. Text does not wrap or ellipsise on its own and nothing centres
// itself, so this package provides the metrics to place each run by hand.
//
// That is only workable because the whole design is set in one monospaced face.
// Every glyph in JetBrains Mono advances the same 0.6em, so the width of a run
// is exact arithmetic rather than a guess, and wrapping and truncation land on
// the same pixel the browser would have chosen.
package svgx

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Rabenherz112/Rabenherz112/internal/fonts"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// Metrics of JetBrains Mono, in em, read from the font itself (unitsPerEm 1000).
const (
	// Advance is the width of every glyph. The face is monospaced, so this one
	// number gives the width of any string.
	Advance = 0.6

	// Ascent and Descent are the font's own vertical extents. Their sum is the
	// line box a browser uses for line-height:normal.
	Ascent  = 1.02
	Descent = 0.30

	// CapHeight is the height of a capital letter above the baseline. Optical
	// centring uses this rather than the full line box, which sits too low.
	CapHeight = 0.73

	// NormalLineHeight is what line-height:normal resolves to for this face.
	NormalLineHeight = Ascent + Descent
)

// Ellipsis is appended to runs cut short by Truncate.
const Ellipsis = "…"

// TextWidth returns the rendered width of s at the given font size, including
// letter spacing. Browsers add the spacing after every glyph, the last one
// included, and this matches that so centred runs line up.
func TextWidth(s string, size, spacing float64) float64 {
	n := float64(len([]rune(s)))
	return n * (size*Advance + spacing)
}

// FitChars returns how many characters fit in width at the given size.
func FitChars(width, size, spacing float64) int {
	per := size*Advance + spacing
	if per <= 0 {
		return 0
	}
	return int(width / per)
}

// FirstBaseline returns the baseline of the first line of a text block whose
// box starts at top, laid out the way CSS would: the leading left over from
// line-height is split evenly above and below the line box.
func FirstBaseline(top, lineHeight, size float64) float64 {
	halfLeading := (lineHeight - size*NormalLineHeight) / 2
	return top + halfLeading + size*Ascent
}

// MiddleBaseline returns the baseline that optically centres a single line on
// centerY, balancing the cap height rather than the full line box.
func MiddleBaseline(centerY, size float64) float64 {
	return centerY + size*CapHeight/2
}

// Wrap breaks s into lines that each fit within maxWidth, splitting on spaces.
// A word longer than the line is placed on its own line rather than broken.
func Wrap(s string, size, spacing, maxWidth float64) []string {
	limit := FitChars(maxWidth, size, spacing)
	if limit <= 0 {
		return nil
	}
	var lines []string
	var line []rune
	for _, word := range strings.Fields(s) {
		w := []rune(word)
		switch {
		case len(line) == 0:
			line = w
		case len(line)+1+len(w) <= limit:
			line = append(append(line, ' '), w...)
		default:
			lines = append(lines, string(line))
			line = w
		}
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

// Truncate shortens s to fit maxWidth, ending with an ellipsis when it had to
// cut. It replaces the CSS text-overflow that an SVG cannot do for itself.
func Truncate(s string, size, spacing, maxWidth float64) string {
	limit := FitChars(maxWidth, size, spacing)
	r := []rune(s)
	if limit <= 0 {
		return ""
	}
	if len(r) <= limit {
		return s
	}
	if limit == 1 {
		return Ellipsis
	}
	return strings.TrimRight(string(r[:limit-1]), " ") + Ellipsis
}

// Text describes how one run is drawn. The zero value is 400-weight text at
// the theme's body colour, anchored at the start.
type Text struct {
	Size    float64
	Fill    string
	Weight  int     // 400 when zero
	Anchor  string  // "", "middle" or "end"
	Spacing float64 // letter-spacing, in px
	Opacity float64 // 1 when zero
}

// A Path is one path of a mark that carries its own colour, as opposed to
// being filled with a colour the card chooses.
type Path struct {
	Fill string
	D    string
}

// A Card accumulates drawing commands and renders them as one SVG document.
//
// Content is written first and the height supplied to Render afterwards, since
// most cards only know how tall they are once their contents are laid out.
type Card struct {
	W     float64
	Theme theme.Theme

	body strings.Builder
	css  strings.Builder
}

// NewCard starts a card width wide in the given palette.
func NewCard(width float64, t theme.Theme) *Card {
	return &Card{W: width, Theme: t}
}

// num formats a coordinate without trailing zeros, to keep the files small.
// Hundredths of a pixel is finer than anything here needs.
func num(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

// ratio formats a scale factor. These are far smaller than one -- fitting a
// 4875-unit viewBox into 30px is a factor of 0.00615 -- so rounding them like a
// coordinate would round most of the value away. Significant digits are what
// matter here, not decimal places.
func ratio(v float64) string {
	return fmt.Sprintf("%.6g", v)
}

var escaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
)

func esc(s string) string { return escaper.Replace(s) }

// Text draws a single run with its baseline at y.
func (c *Card) Text(x, y float64, s string, o Text) {
	if s == "" {
		return
	}
	fill := o.Fill
	if fill == "" {
		fill = c.Theme.Body
	}
	fmt.Fprintf(&c.body, `<text x="%s" y="%s" font-size="%s" fill="%s"`,
		num(x), num(y), num(o.Size), fill)
	if o.Weight != 0 && o.Weight != 400 {
		fmt.Fprintf(&c.body, ` font-weight="%d"`, o.Weight)
	}
	if o.Anchor != "" {
		fmt.Fprintf(&c.body, ` text-anchor="%s"`, o.Anchor)
	}
	if o.Spacing != 0 {
		fmt.Fprintf(&c.body, ` letter-spacing="%s"`, num(o.Spacing))
	}
	if o.Opacity != 0 && o.Opacity != 1 {
		fmt.Fprintf(&c.body, ` opacity="%s"`, num(o.Opacity))
	}
	fmt.Fprintf(&c.body, ">%s</text>", esc(s))
}

// TextMiddle draws a run centred on x. Letter spacing is added after the last
// glyph too, so the run is nudged back by half a step to sit truly centred.
func (c *Card) TextMiddle(x, y float64, s string, o Text) {
	o.Anchor = "middle"
	c.Text(x-o.Spacing/2, y, s, o)
}

// TextEnd draws a run ending at x.
func (c *Card) TextEnd(x, y float64, s string, o Text) {
	o.Anchor = "end"
	c.Text(x-o.Spacing, y, s, o)
}

// Rect fills a rectangle.
func (c *Card) Rect(x, y, w, h float64, fill string) {
	if w <= 0 || h <= 0 {
		return
	}
	fmt.Fprintf(&c.body, `<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`,
		num(x), num(y), num(w), num(h), fill)
}

// RoundRect fills a rectangle with rounded corners.
func (c *Card) RoundRect(x, y, w, h, r float64, fill string) {
	c.RoundRectOpacity(x, y, w, h, r, fill, 1)
}

// RoundRectOpacity is RoundRect with a fill opacity, for marks that should sit
// back from the ones around them.
func (c *Card) RoundRectOpacity(x, y, w, h, r float64, fill string, opacity float64) {
	if w <= 0 || h <= 0 {
		return
	}
	fmt.Fprintf(&c.body, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"`,
		num(x), num(y), num(w), num(h), num(r), fill)
	if opacity != 1 {
		fmt.Fprintf(&c.body, ` opacity="%s"`, num(opacity))
	}
	c.body.WriteString(`/>`)
}

// Divider draws the hairline used between sections of a card.
func (c *Card) Divider(x, y, w float64) {
	c.Rect(x, y, w, 1, c.Theme.Divider)
}

// StrokeRoundRect outlines a rectangle with rounded corners. The half-pixel
// offset puts a 1px stroke on the pixel grid instead of straddling two.
func (c *Card) StrokeRoundRect(x, y, w, h, r float64, stroke string) {
	fmt.Fprintf(&c.body,
		`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="none" stroke="%s"/>`,
		num(x+0.5), num(y+0.5), num(w-1), num(h-1), num(r), stroke)
}

// Mark draws a single-colour vector mark, scaling its square viewBox down to
// size and placing its top-left corner at (x, y).
func (c *Card) Mark(x, y, size, viewBox float64, paths []string, fill string) {
	fmt.Fprintf(&c.body, `<g transform="translate(%s %s) scale(%s)" fill="%s">`,
		num(x), num(y), ratio(size/viewBox), fill)
	for _, d := range paths {
		fmt.Fprintf(&c.body, `<path d="%s"/>`, d)
	}
	c.body.WriteString(`</g>`)
}

// Glyph draws artwork whose viewBox is not square, scaled to the given width
// and placed with its top-left corner at (x, y). Mark is the square case; this
// is for artwork with its own proportions, such as the bird.
func (c *Card) Glyph(x, y, width, vbW, vbH float64, paths []string, fill string, opacity float64) {
	fmt.Fprintf(&c.body, `<g transform="translate(%s %s) scale(%s)" fill="%s"`,
		num(x), num(y), ratio(width/vbW), fill)
	if opacity != 1 {
		fmt.Fprintf(&c.body, ` opacity="%s"`, num(opacity))
	}
	c.body.WriteString(`>`)
	for _, d := range paths {
		fmt.Fprintf(&c.body, `<path d="%s"/>`, d)
	}
	c.body.WriteString(`</g>`)
}

// GlyphHeight returns how tall a Glyph of the given width will be.
func GlyphHeight(width, vbW, vbH float64) float64 { return width * vbH / vbW }

// ColorMark draws a mark in its own colours. class is optional and attaches the
// mark to a rule added with CSS, which is how the waving hand is animated.
//
// A class goes on a group nested inside the positioning one, never the same
// group. A CSS transform replaces the transform attribute outright rather than
// composing with it, so an animation sharing a group with the translate and
// scale would throw the mark to the origin at full size.
func (c *Card) ColorMark(x, y, size, viewBox float64, paths []Path, class string) {
	fmt.Fprintf(&c.body, `<g transform="translate(%s %s) scale(%s)">`,
		num(x), num(y), ratio(size/viewBox))
	if class != "" {
		fmt.Fprintf(&c.body, `<g class="%s">`, class)
	}
	for _, p := range paths {
		fmt.Fprintf(&c.body, `<path fill="%s" d="%s"/>`, p.Fill, p.D)
	}
	if class != "" {
		c.body.WriteString(`</g>`)
	}
	c.body.WriteString(`</g>`)
}

// CSS adds a rule to the card's stylesheet.
//
// Declarative CSS runs inside an SVG loaded through an <img> tag, animations
// included, which is the one bit of life a card can have. Script does not run
// there, so anything moving has to be expressed this way.
func (c *Card) CSS(rule string) { c.css.WriteString(rule) }

// Image embeds raster data inline. GitHub blocks external references from
// inside a card, so anything shown has to travel with the file.
func (c *Card) Image(x, y, w, h float64, mime string, data []byte) {
	fmt.Fprintf(&c.body,
		`<image x="%s" y="%s" width="%s" height="%s" preserveAspectRatio="xMidYMid meet" href="data:%s;base64,%s"/>`,
		num(x), num(y), num(w), num(h), mime, base64.StdEncoding.EncodeToString(data))
}

// Raw appends already-formed SVG markup.
func (c *Card) Raw(s string) { c.body.WriteString(s) }

// A Gutter is transparent space added outside a card's frame, widening the
// document without moving anything the card drew relative to its own box.
//
// It exists so that two half-width cards can tile a row exactly. Markdown
// places them as two inline images, and any gap written between the images
// costs the row that much width, leaving its right edge short of the
// full-width cards around it. Carrying the gap inside the images instead lets
// each one be exactly half the row.
type Gutter struct{ Left, Right float64 }

// Render closes the card at the given height and returns the SVG document.
//
// The card paints no background. GitHub has four themes but only tells a card
// whether it is on a light or a dark one, so the two variants stay transparent
// and let the real page colour show through, which reads correctly on the
// dimmed and high-contrast themes as well.
func (c *Card) Render(h float64) []byte {
	return c.render(h, true, true, Gutter{})
}

// RenderGutter is Render with transparent space outside the card's frame.
func (c *Card) RenderGutter(h float64, g Gutter) []byte {
	return c.render(h, true, true, g)
}

// RenderFrameless is Render without the card outline, for pieces drawn without
// a box around them.
func (c *Card) RenderFrameless(h float64) []byte {
	return c.render(h, false, true, Gutter{})
}

// RenderBare is RenderFrameless without the embedded font, for a card that
// draws no text. The font is 42KB once encoded, and the link tiles are only
// icons, so carrying it would multiply their size forty-fold for nothing.
func (c *Card) RenderBare(h float64) []byte {
	return c.render(h, false, false, Gutter{})
}

func (c *Card) render(h float64, frame, withFont bool, g Gutter) []byte {
	w := c.W + g.Left + g.Right

	var out strings.Builder
	fmt.Fprintf(&out,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" fill="none">`,
		num(w), num(h), num(w), num(h))

	if style := c.styles(withFont); style != "" {
		fmt.Fprintf(&out, `<defs><style>%s</style></defs>`, style)
	}

	// Every command the card recorded is in the card's own coordinates, so a
	// gutter on the left is one translate rather than an offset on each of
	// them. A gutter on the right only widens the document.
	if g.Left != 0 {
		fmt.Fprintf(&out, `<g transform="translate(%s 0)">`, num(g.Left))
	}

	if frame {
		// The border sits on a half pixel so a 1px stroke lands on the pixel
		// grid instead of straddling two.
		fmt.Fprintf(&out,
			`<rect x="0.5" y="0.5" width="%s" height="%s" rx="6" fill="none" stroke="%s"/>`,
			num(c.W-1), num(h-1), c.Theme.Border)
	}

	out.WriteString(c.body.String())

	if g.Left != 0 {
		out.WriteString(`</g>`)
	}
	out.WriteString(`</svg>`)
	return []byte(out.String())
}

// styles builds the card's stylesheet: the embedded face, and whatever rules
// the card added through CSS.
//
// The face travels with the file because GitHub's image proxy blocks the font
// request an external @font-face would make, which would drop the card back to
// the viewer's default monospace and break every measurement this package makes.
func (c *Card) styles(withFont bool) string {
	var s strings.Builder
	if withFont {
		fmt.Fprintf(&s, `@font-face{font-family:'JBM';src:url(data:font/woff2;base64,%s)format('woff2');font-weight:400 800;font-style:normal}text{font-family:'JBM',ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;white-space:pre}`,
			fonts.JetBrainsMonoBase64())
	}
	s.WriteString(c.css.String())
	return s.String()
}
