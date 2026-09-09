package cards

import (
	"fmt"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// AniListCommand heads the card.
const AniListCommand = "$ anilist --manga --stats"

const (
	aniStatSize = 24.0

	aniCoverW    = 52.0
	aniCoverGap  = 12.0
	aniCoverAR   = 3.0 / 2.0 // covers are 2:3
	aniColGap    = 20.0
	aniTitleSize = 12.0
	aniTitleLine = aniTitleSize * 1.4
	aniMetaSize  = 11.0

	aniHeadingSize = 11.0
)

// AniList draws the manga totals and the series currently being read.
func AniList(t theme.Theme, data model.AniList, note string) Built {
	c := svgx.NewCard(WidthFull, t)
	content := WidthFull - 2*PadX

	y := labelWithNote(c, PadTop, AniListCommand, note, 20)

	known := data.Known()
	y = statRow(c, y, []stat{
		{Value: figure(known, comma(data.SeriesTracked)), Label: "SERIES TRACKED"},
		{Value: figure(known, comma(data.ChaptersRead)), Label: "CHAPTERS READ"},
		{Value: figure(known, comma(data.VolumesRead)), Label: "VOLUMES READ"},
		{Value: figure(known, fmt.Sprintf("%.1f", data.MeanScore)), Label: "MEAN SCORE"},
	}, aniStatSize)

	y += 22
	c.Divider(PadX, y, content)
	y += 1 + 20

	c.Text(PadX, svgx.FirstBaseline(y, lineBox(aniHeadingSize), aniHeadingSize),
		"READING NOW", svgx.Text{
			Size:    aniHeadingSize,
			Fill:    t.Muted,
			Spacing: aniHeadingSize * 0.18,
		})
	y += lineBox(aniHeadingSize) + 14

	if len(data.Reading) == 0 {
		y = emptyNote(c, y, "nothing in progress")
	} else {
		y += drawReading(c, y, content, data.Reading)
	}

	return Built{Card: c, Height: y + PadBottom}
}

// drawReading lays the in-progress series out in three columns and returns the
// height it used.
func drawReading(c *svgx.Card, top, content float64, media []model.Media) float64 {
	const cols = 3
	if len(media) == 0 {
		return 0
	}

	colW := (content - aniColGap*(cols-1)) / cols
	textW := colW - aniCoverW - aniCoverGap
	coverH := aniCoverW * aniCoverAR

	var tallest float64
	for i, m := range media {
		if i == cols {
			break
		}
		x := PadX + float64(i)*(colW+aniColGap)
		drawCover(c, x, top, aniCoverW, coverH, m)

		tx := x + aniCoverW + aniCoverGap
		lines := svgx.Wrap(m.Title, aniTitleSize, 0, textW)
		for j, line := range lines {
			base := svgx.FirstBaseline(top+float64(j)*aniTitleLine, aniTitleLine, aniTitleSize)
			c.Text(tx, base, line, svgx.Text{Size: aniTitleSize, Fill: c.Theme.Strong})
		}

		metaTop := top + aniTitleLine*float64(len(lines)) + 3
		c.Text(tx, svgx.FirstBaseline(metaTop, lineBox(aniMetaSize), aniMetaSize),
			m.Meta, svgx.Text{Size: aniMetaSize, Fill: c.Theme.Muted})

		tallest = max(tallest, max(coverH, metaTop-top+lineBox(aniMetaSize)))
	}
	return tallest
}

// drawCover places a cover image, falling back to the design's hatched
// placeholder when the artwork could not be fetched.
func drawCover(c *svgx.Card, x, y, w, h float64, m model.Media) {
	if len(m.Cover) > 0 {
		c.Image(x, y, w, h, m.CoverMIME, m.Cover)
	} else {
		c.RoundRect(x, y, w, h, 3, c.Theme.Divider)
	}
	// The outline goes on top so it reads as a frame around the art rather than
	// a box behind it.
	c.Raw(fmt.Sprintf(
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="3" fill="none" stroke="%s"/>`,
		x+0.5, y+0.5, w-1, h-1, c.Theme.Border))
}
