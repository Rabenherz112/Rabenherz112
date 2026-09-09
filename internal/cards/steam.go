package cards

import (
	"fmt"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// SteamCommand heads the card.
const SteamCommand = "$ steam --recent"

const (
	steamRowSize = 12.0
	steamRowGap  = 16.0
	steamBarH    = 3.0
	steamBarGap  = 7.0

	// The store icon sits left of the name, the way it does in the Steam
	// client's own recently-played list.
	steamIconSize = 28.0
	steamIconGap  = 12.0
	steamIconR    = 3.0
)

// Steam draws what has been played in the last two weeks.
func Steam(t theme.Theme, data model.Steam) Built {
	c := svgx.NewCard(WidthHalf, t)
	content := WidthHalf - 2*PadX

	note := ""
	if data.Known() && data.OwnedGames > 0 {
		note = fmt.Sprintf("%s games owned", comma(data.OwnedGames))
	}
	y := labelWithNote(c, PadTop, SteamCommand, note, LabelGap)

	if !data.Known() || len(data.Recent) == 0 {
		return Built{Card: c, Height: emptyNote(c, y, "nothing played in the last two weeks") + PadBottom}
	}

	// Bars are relative to the most-played game rather than to the fortnight,
	// so the card reads as a ranking. An absolute scale would leave every bar
	// near zero in a quiet week.
	peak := 0
	for _, g := range data.Recent {
		peak = max(peak, g.TwoWeekMinutes)
	}

	textX := PadX + steamIconSize + steamIconGap
	textW := content - steamIconSize - steamIconGap

	for i, g := range data.Recent {
		if i > 0 {
			y += steamRowGap
		}

		drawGameIcon(c, PadX, y, g)

		base := svgx.FirstBaseline(y, lineBox(steamRowSize), steamRowSize)
		meta := hoursMinutes(time.Duration(g.TwoWeekMinutes)*time.Minute) + " past 2 weeks"
		metaW := svgx.TextWidth(meta, steamRowSize, 0)

		c.Text(textX, base, svgx.Truncate(g.Name, steamRowSize, 0, textW-metaW-12),
			svgx.Text{Size: steamRowSize, Fill: t.Strong})
		c.TextEnd(WidthHalf-PadX, base, meta, svgx.Text{Size: steamRowSize, Fill: t.Muted})

		barY := y + lineBox(steamRowSize) + steamBarGap
		c.Rect(textX, barY, textW, steamBarH, t.Track)
		if peak > 0 {
			c.Rect(textX, barY, textW*float64(g.TwoWeekMinutes)/float64(peak), steamBarH, t.Accent)
		}

		// The icon is taller than the name and bar together, so the row is as
		// tall as whichever wins.
		y += max(steamIconSize, barY+steamBarH-y)
	}

	return Built{Card: c, Height: y + PadBottom}
}

// drawGameIcon places a game's store icon, falling back to an empty square when
// the artwork could not be fetched.
func drawGameIcon(c *svgx.Card, x, y float64, g model.Game) {
	if len(g.Icon) > 0 {
		c.Image(x, y, steamIconSize, steamIconSize, g.IconMIME, g.Icon)
	} else {
		c.RoundRect(x, y, steamIconSize, steamIconSize, steamIconR, c.Theme.Divider)
	}
	c.StrokeRoundRect(x, y, steamIconSize, steamIconSize, steamIconR, c.Theme.Border)
}
