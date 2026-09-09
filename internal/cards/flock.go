package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
)

// The flock is the profile's one piece of ornament, and the only drawing here
// that is not somebody else's logo. Rabenherz is a raven's heart, so a bird in
// flight is the obvious motif, and it turns up twice: crossing the top of the
// header, and beside the number on the view counter.
//
// There is only one bird. It is drawn seven times at falling size and opacity,
// which is what makes a flock read as depth rather than as a row of stamps.

// Bird is a crow in side profile, gliding right with the near wing raised,
// drawn in a 26 x 16 box.
//
// Side-on rather than head-on: a bird seen from the front is two symmetric
// strokes and reads as a gull, where the profile carries a head, a body and a
// tail, and reads as the crow the name is about.
const (
	BirdWidth  = 26.0
	BirdHeight = 16.0
)

// BirdPaths is the whole silhouette as one outline. It keeps a solid body
// rather than tapering to hairlines, because the last bird in the flock is
// drawn eleven pixels wide and detail disappears well before that.
var BirdPaths = []string{
	"M25.8 7.9C24.6 7.2 23.5 6.7 22.4 6.4 21.6 6.1 20.7 6 19.8 6.05 18.3 6.15 16.9 6.5 15.6 7.05 13.1 4.2 10 1.8 6.3 0 8.5 3.2 10.8 6.1 13.3 8.6 10.3 8.8 7.3 9.05 4.3 9.35 2.9 9.5 1.4 9.65 0 9.85 1.5 10.65 3.1 11.25 4.8 11.65 8.2 12.45 11.8 12.55 15.2 11.95 17.6 11.55 19.8 10.65 21.7 9.35 23.1 8.95 24.5 8.5 25.8 7.9Z",
}

// A bird is one member of a flock: where it sits relative to the flock's
// top-left corner, how wide it is drawn, and how far back it sits.
type bird struct {
	X, Y    float64
	Width   float64
	Opacity float64
}

// headerFlock crosses the top right of the header card. The birds lead with the
// largest and trail off to the right, and none of them share a baseline: a
// flock in a straight line looks like a ruler, not like birds.
var headerFlock = []bird{
	{X: 0, Y: 6, Width: 26, Opacity: 0.55},
	{X: 40, Y: 1, Width: 24, Opacity: 0.47},
	{X: 76, Y: 10, Width: 22, Opacity: 0.40},
	{X: 110, Y: 3, Width: 20, Opacity: 0.32},
	{X: 141, Y: 12, Width: 17, Opacity: 0.26},
	{X: 167, Y: 5, Width: 13, Opacity: 0.20},
	{X: 187, Y: 14, Width: 11, Opacity: 0.15},
}

// flockWidth is how much room the header flock needs.
const flockWidth = 198.0

// drawFlock places a flock with its top-left corner at (x, y).
func drawFlock(c *svgx.Card, x, y float64, flock []bird, fill string) {
	for _, b := range flock {
		c.Glyph(x+b.X, y+b.Y, b.Width, BirdWidth, BirdHeight, BirdPaths, fill, b.Opacity)
	}
}
