package cards

import (
	"math"

	"github.com/Rabenherz112/Rabenherz112/internal/linguist"
	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// LanguagesCommand heads the card.
const LanguagesCommand = "$ lang --stat --last 12mo"

const (
	langRowSize = 12.0
	langRowGap  = 11.0
	langNameW   = 88.0
	langPctW    = 44.0
	langColGap  = 12.0

	// The design draws the bars as a run of block characters at 12px with a
	// pixel of negative tracking, so each block steps 6.2px and they close up
	// into a solid bar. JetBrains Mono has no block glyph, so the bar is drawn
	// as a rectangle of the width that run would have occupied. One block
	// stands for two percent, which is the scale the design uses.
	langBlockStep   = langRowSize*svgx.Advance - 1
	langBlockWidth  = langRowSize * svgx.Advance
	langPctPerBlock = 2.0
)

// Languages draws the share of coding time per language over the last year.
func Languages(t theme.Theme, data model.Languages) Built {
	c := svgx.NewCard(WidthHalf, t)
	content := WidthHalf - 2*PadX

	y := label(c, PadTop, LanguagesCommand)

	barX := PadX + langNameW + langColGap
	barMax := content - langNameW - langColGap - langColGap - langPctW
	nameMax := langNameW - 2 // a hair of clearance before the bar starts

	for i, l := range data.Items {
		if i > 0 {
			y += langRowGap
		}
		rowH := lineBox(langRowSize)
		center := y + rowH/2
		base := svgx.MiddleBaseline(center, langRowSize)

		c.Text(PadX, base, svgx.Truncate(l.Name, langRowSize, 0, nameMax),
			svgx.Text{Size: langRowSize, Fill: t.Body})

		blocks := math.Max(1, math.Round(l.Pct/langPctPerBlock))
		w := math.Min((blocks-1)*langBlockStep+langBlockWidth, barMax)
		c.Rect(barX, center-langRowSize/2, w, langRowSize, linguist.Color(l.Name))

		c.TextEnd(WidthHalf-PadX, base, pct(l.Pct),
			svgx.Text{Size: langRowSize, Fill: t.Muted})

		y += rowH
	}

	return Built{Card: c, Height: y + PadBottom}
}
