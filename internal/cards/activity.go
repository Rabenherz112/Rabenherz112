package cards

import (
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// ActivityCommand heads the card. The count matches how many lines it holds.
const ActivityCommand = "$ git log --author=rabenherz --review -n 10"

const (
	actRowSize = 12.0
	actRowGap  = 10.0
	actColGap  = 14.0

	actVerbW = 82.0
	actNumW  = 52.0
	actWhenW = 40.0
)

// Activity draws the most recent reviews and pull requests.
//
// Nothing on this card is a link. A card is one flat image to GitHub, so a
// per-row link is not possible; the README carries a single "see the full
// activity feed" link underneath instead.
func Activity(t theme.Theme, data model.Activity, note string, now time.Time) Built {
	c := svgx.NewCard(WidthFull, t)
	content := WidthFull - 2*PadX

	y := labelWithNote(c, PadTop, ActivityCommand, note, LabelGap)

	numX := PadX + actVerbW + actColGap
	repoX := numX + actNumW + actColGap
	repoW := content - actVerbW - actNumW - actWhenW - 3*actColGap

	for i, e := range data.Items {
		if i > 0 {
			y += actRowGap
		}
		base := svgx.FirstBaseline(y, lineBox(actRowSize), actRowSize)

		c.Text(PadX, base, e.Verb, svgx.Text{Size: actRowSize, Fill: t.Verb(e.Verb)})

		// Pull requests and issues give this column a short "#3038", but a
		// release gives it a tag, and a tag can be any length at all.
		c.Text(numX, base, svgx.Truncate(e.Num, actRowSize, 0, actNumW),
			svgx.Text{Size: actRowSize, Fill: t.Accent})
		c.Text(repoX, base, svgx.Truncate(e.Repo, actRowSize, 0, repoW),
			svgx.Text{Size: actRowSize, Fill: t.Body})
		c.TextEnd(WidthFull-PadX, base, since(e.When, now),
			svgx.Text{Size: actRowSize, Fill: t.Faint})

		y += lineBox(actRowSize)
	}

	return Built{Card: c, Height: y + PadBottom}
}
