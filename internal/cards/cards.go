// Package cards lays out and draws each panel of the profile README.
//
// The geometry here is a direct translation of the design's CSS. Every card is
// a vertical stack: a "$ command" label, some content, and padding. Because an
// SVG has no layout engine, each block computes its own height and hands the
// cursor to the next one, and the card's total height falls out of the last
// one. That is why the draw functions return a y coordinate.
//
// Widths are fixed. A full-width card is drawn at 880 and placed at 100%; the
// paired cards are drawn at 432, rendered into a 440-wide document by
// RenderPair and placed at 50% each, so two sit side by side the way the
// design has them and the row is exactly as wide as a full-width card.
package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// Card widths, in the SVG's own coordinates.
const (
	WidthFull = 880.0
	WidthHalf = 432.0
)

// Padding and the label block, shared by every card except the header, which
// is roomier.
const (
	PadX      = 24.0
	PadTop    = 22.0
	PadBottom = 22.0

	LabelSize    = 11.0
	LabelSpacing = LabelSize * 0.18
	LabelGap     = 18.0 // space between the label and the content below it

	// NoteSize is the small print in a card's top-right corner and footer.
	NoteSize = 10.0
)

// lineBox is the height of one line of text at the given size, the same figure
// a browser would use for line-height:normal.
func lineBox(size float64) float64 { return size * svgx.NormalLineHeight }

// label draws a card's "$ command" heading at top and returns the y the
// content below it starts at.
func label(c *svgx.Card, top float64, text string) float64 {
	c.Text(PadX, svgx.FirstBaseline(top, lineBox(LabelSize), LabelSize), text, svgx.Text{
		Size:    LabelSize,
		Fill:    c.Theme.Muted,
		Spacing: LabelSpacing,
	})
	return top + lineBox(LabelSize) + LabelGap
}

// labelWithNote is label plus a caption set flush right on the same baseline.
func labelWithNote(c *svgx.Card, top float64, text, note string, gap float64) float64 {
	base := svgx.FirstBaseline(top, lineBox(LabelSize), LabelSize)
	c.Text(PadX, base, text, svgx.Text{
		Size:    LabelSize,
		Fill:    c.Theme.Muted,
		Spacing: LabelSpacing,
	})
	c.TextEnd(c.W-PadX, base, note, svgx.Text{
		Size: NoteSize,
		Fill: c.Theme.Faint,
	})
	return top + lineBox(LabelSize) + gap
}

// spaceBetween spreads items of the given widths across a row, first flush
// left and last flush right, matching CSS justify-content:space-between. It
// returns the left edge of each item.
func spaceBetween(left, width float64, widths []float64) []float64 {
	xs := make([]float64, len(widths))
	if len(widths) == 0 {
		return xs
	}
	if len(widths) == 1 {
		xs[0] = left
		return xs
	}
	var total float64
	for _, w := range widths {
		total += w
	}
	gap := (width - total) / float64(len(widths)-1)
	if gap < 0 {
		gap = 0
	}
	x := left
	for i, w := range widths {
		xs[i] = x
		x += w + gap
	}
	return xs
}

// NoData is what stands in for a figure that has never been measured.
//
// A source that has never worked must not be able to look like one that has.
// Zeroes read as a real measurement, and seeded placeholder numbers are worse
// still, because nothing about them says they are made up.
const NoData = "—"

// figure returns a value, or the no-data dash when the section it came from has
// never been fetched.
func figure(known bool, s string) string {
	if !known {
		return NoData
	}
	return s
}

// emptyNote writes a card's empty state and returns the y below it.
func emptyNote(c *svgx.Card, top float64, text string) float64 {
	c.Text(PadX, svgx.FirstBaseline(top, lineBox(NoteSize), NoteSize), text,
		svgx.Text{Size: NoteSize, Fill: c.Theme.Faint})
	return top + lineBox(NoteSize)
}

// A stat is one figure in a card's headline row: a large value over a small
// spaced-out caption.
type stat struct {
	Value string
	Label string
	// Fill overrides the value colour. Empty means the theme's strong text.
	Fill string
}

// statRow draws a row of figures spread across the card's content width and
// returns the y below it.
func statRow(c *svgx.Card, top float64, stats []stat, valueSize float64) float64 {
	const (
		labelSize    = 10.0
		labelSpacing = labelSize * 0.12
		labelGap     = 4.0
	)

	widths := make([]float64, len(stats))
	for i, s := range stats {
		vw := svgx.TextWidth(s.Value, valueSize, 0)
		lw := svgx.TextWidth(s.Label, labelSize, labelSpacing)
		widths[i] = max(vw, lw)
	}
	xs := spaceBetween(PadX, c.W-2*PadX, widths)

	valueBase := svgx.FirstBaseline(top, lineBox(valueSize), valueSize)
	labelTop := top + lineBox(valueSize) + labelGap
	labelBase := svgx.FirstBaseline(labelTop, lineBox(labelSize), labelSize)

	for i, s := range stats {
		fill := s.Fill
		if fill == "" {
			fill = c.Theme.Strong
		}
		c.Text(xs[i], valueBase, s.Value, svgx.Text{Size: valueSize, Fill: fill})
		c.Text(xs[i], labelBase, s.Label, svgx.Text{
			Size:    labelSize,
			Fill:    c.Theme.Muted,
			Spacing: labelSpacing,
		})
	}
	return labelTop + lineBox(labelSize)
}

// A Built card has been drawn but not yet closed, so its height can still be
// adjusted to match the card beside it.
type Built struct {
	Card   *svgx.Card
	Height float64
}

// Render closes the card at its natural height.
func (b Built) Render() []byte { return b.Card.Render(b.Height) }

// PairGutter is the transparent space a half-width card carries on the side
// that faces the other half of its row. Two of them make up the gap the design
// leaves between the boxes, which is what WidthFull and WidthHalf already
// imply: 880 is 432, a 16 gap, and 432 again.
const PairGutter = (WidthFull - 2*WidthHalf) / 2

// RenderPair closes two half-width cards as one row.
//
// Two things have to be right for a row of two to sit squarely under the
// full-width cards around it.
//
// Both halves have to be the same height. Markdown scales each image by its
// own aspect ratio, so at their natural heights their bottom edges would not
// line up. Squaring them off is also what the design's CSS grid does.
//
// And the two images have to tile the row exactly. Each is rendered a full
// WidthFull/2 wide and placed at width="50%", carrying the gap between the
// boxes as transparent space on its inner side. Writing that gap between the
// images instead costs the row the width of the gap, so its right edge stops
// short of every full-width card on the page -- and there is no way to write
// one in markdown that does not also let the two wrap onto separate lines once
// their widths sum close enough to the full width.
func RenderPair(a, b Built) (left, right []byte) {
	h := max(a.Height, b.Height)
	return a.Card.RenderGutter(h, svgx.Gutter{Right: PairGutter}),
		b.Card.RenderGutter(h, svgx.Gutter{Left: PairGutter})
}

// Theme is re-exported so callers building a set of cards need only this
// package's import.
type Theme = theme.Theme
