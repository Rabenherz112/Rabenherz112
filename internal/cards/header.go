package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/icons"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
	"github.com/Rabenherz112/Rabenherz112/internal/wordmark"
)

// Header content. It never changes, so it lives here rather than in the
// snapshot: there is nothing to fetch and nothing to go stale.
const (
	HeaderCommand = "$ whoami --verbose"

	// The double spaces around the first separator are in the design, setting
	// the handle apart from the list of interests that follows it.
	HeaderTagline = "aka Lp_Zombie  ·  IT security · Gamer · Selfhoster · Manga reader"

	HeaderBio = "I'm an IT specialist working at Serviceware SE, where I handle the security of the IT systems to ensure smooth operations. Off the clock, I run a small fleet of V-servers, Raspberry Pis, and a Synology NAS hosting web services and other playground projects. I also enjoy gaming, especially roguelike games, and reading manga, of which I have an extensive collection that I'm proud of."
)

// Header geometry. This card is padded more generously than the rest.
const (
	headerPadX      = 32.0
	headerPadTop    = 30.0
	headerPadBottom = 28.0

	// The wordmark is centred and never drawn wider than this.
	headerMarkMaxWidth = 520.0

	// The waving hand greets alongside the whoami, where the old profile opened
	// with "Hey there!" and a waving GIF.
	headerWaveSize = 22.0

	// The bio is centred in a measure narrower than the card, so the lines stay
	// short enough to read.
	headerBioMeasure    = 660.0
	headerBioSize       = 13.0
	headerBioLineHeight = headerBioSize * 1.85
)

// Header draws the wordmark, tagline and bio.
func Header(t theme.Theme, mark wordmark.Prepared) Built {
	c := svgx.NewCard(WidthFull, t)
	center := WidthFull / 2

	// The command label is the one thing on this card set flush left; the rest
	// is centred under the wordmark.
	y := headerPadTop
	c.Text(headerPadX, svgx.FirstBaseline(y, lineBox(12), 12), HeaderCommand, svgx.Text{
		Size: 12,
		Fill: t.Muted,
	})

	// The hand waves next to the greeting it belongs with, rather than in the
	// corner, which the flock now crosses.
	drawWave(c, headerPadX+svgx.TextWidth(HeaderCommand, 12, 0)+10, y-6)

	drawFlock(c, WidthFull-headerPadX-flockWidth, headerPadTop-4, headerFlock, t.Muted)

	y += lineBox(12) + 16

	markW := min(headerMarkMaxWidth, WidthFull-2*headerPadX)
	markH := markW * float64(mark.Height) / float64(mark.Width)
	c.Image(center-markW/2, y, markW, markH, wordmark.MIME, mark.PNG)
	y += markH

	y += 10
	const taglineSize = 13.0
	c.TextMiddle(center, svgx.FirstBaseline(y, lineBox(taglineSize), taglineSize),
		HeaderTagline, svgx.Text{
			Size:    taglineSize,
			Fill:    t.Muted,
			Spacing: taglineSize * 0.05,
		})
	y += lineBox(taglineSize)

	y += 22
	c.Divider(headerPadX, y, WidthFull-2*headerPadX)
	y += 1 + 18

	bio := svgx.Wrap(HeaderBio, headerBioSize, 0, headerBioMeasure)
	for i, line := range bio {
		base := svgx.FirstBaseline(y+float64(i)*headerBioLineHeight,
			headerBioLineHeight, headerBioSize)
		c.TextMiddle(center, base, line, svgx.Text{Size: headerBioSize, Fill: t.Body})
	}
	y += headerBioLineHeight * float64(len(bio))

	return Built{Card: c, Height: y + headerPadBottom}
}

// drawWave places the waving hand and sets it going.
//
// A card is a still image to GitHub and script never runs inside one, but
// declarative CSS does, animations included. This is the one thing on the
// profile that moves.
func drawWave(c *svgx.Card, x, y float64) {
	paths := make([]svgx.Path, len(icons.WavingHand.Paths))
	for i, p := range icons.WavingHand.Paths {
		paths[i] = svgx.Path{Fill: p.Fill, D: p.D}
	}
	c.ColorMark(x, y, headerWaveSize, icons.WavingHand.ViewBox, paths, "wave")

	// The hand pivots about its wrist, near the bottom of the artwork, rather
	// than about its centre. fill-box makes the origin relative to the mark's
	// own bounding box instead of the whole card.
	c.CSS(`.wave{transform-box:fill-box;transform-origin:50% 85%;` +
		`animation:wave 2.8s ease-in-out infinite}` +
		`@keyframes wave{0%,55%,100%{transform:rotate(0)}` +
		`62%{transform:rotate(16deg)}70%{transform:rotate(-9deg)}` +
		`78%{transform:rotate(16deg)}86%{transform:rotate(-5deg)}}` +
		// Anyone who has asked their system to stop things moving gets a
		// still hand rather than a permanent wave in the corner of a page.
		`@media(prefers-reduced-motion:reduce){.wave{animation:none}}`)
}
