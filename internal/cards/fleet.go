package cards

import (
	"github.com/Rabenherz112/Rabenherz112/internal/icons"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// FleetCommand heads the card.
const FleetCommand = "$ cat /etc/os-release --fleet"

// os is one of the two operating systems the card describes.
type osEntry struct {
	Name string
	Note string
	Icon icons.Icon
	// Fill is the mark's brand colour, which reads on both backgrounds and so
	// is the same in either theme.
	Fill string
	// XOffset nudges a mark that does not fill its square viewBox.
	XOffset float64
}

var fleetOS = []osEntry{
	{
		Name:    "Debian",
		Note:    "I use it for most of my personal infrastructure. Stable, light, and it never changes.",
		Icon:    icons.Debian,
		Fill:    icons.DebianColor,
		XOffset: icons.DebianXOffset,
	},
	{
		Name: "Windows",
		Note: "Daily driver at work and at home. Where the PowerShell gets written and the games get played.",
		Icon: icons.Windows,
		Fill: icons.WindowsColor,
	},
}

const (
	fleetMarkSize = 30.0
	fleetMarkGap  = 14.0
	fleetColGap   = 24.0

	fleetNameSize = 13.0
	fleetNoteSize = 12.0
	fleetNoteLine = fleetNoteSize * 1.6

	// The toolbox strip wraps across as many rows as it needs. The gap is
	// tighter than the design's, since the strip now carries the whole toolbox
	// rather than a sample of it followed by a line of text.
	fleetToolSize = 24.0
	fleetToolGap  = 14.0
	fleetToolRow  = 14.0
)

// Fleet draws the two operating systems and the toolbox strip. Nothing here is
// fetched: it is a statement about the setup, not a metric.
func Fleet(t theme.Theme) Built {
	c := svgx.NewCard(WidthFull, t)
	content := WidthFull - 2*PadX

	y := label(c, PadTop, FleetCommand)

	colW := (content - fleetColGap) / 2
	textW := colW - fleetMarkSize - fleetMarkGap

	var tallest float64
	for i, os := range fleetOS {
		x := PadX + float64(i)*(colW+fleetColGap)

		// The mark sits a couple of pixels low so its body lines up with the
		// x-height of the name beside it rather than its cap line.
		c.Mark(x+os.XOffset*(fleetMarkSize/os.Icon.ViewBox), y+2,
			fleetMarkSize, os.Icon.ViewBox, os.Icon.Paths, os.Fill)

		tx := x + fleetMarkSize + fleetMarkGap
		c.Text(tx, svgx.FirstBaseline(y, lineBox(fleetNameSize), fleetNameSize),
			os.Name, svgx.Text{Size: fleetNameSize, Fill: t.Strong})

		noteTop := y + lineBox(fleetNameSize) + 4
		lines := svgx.Wrap(os.Note, fleetNoteSize, 0, textW)
		for j, line := range lines {
			base := svgx.FirstBaseline(noteTop+float64(j)*fleetNoteLine,
				fleetNoteLine, fleetNoteSize)
			c.Text(tx, base, line, svgx.Text{Size: fleetNoteSize, Fill: t.Muted})
		}

		h := max(fleetMarkSize+2,
			lineBox(fleetNameSize)+4+fleetNoteLine*float64(len(lines)))
		tallest = max(tallest, h)
	}
	y += tallest

	y += 18
	c.Divider(PadX, y, content)
	y += 1 + 14

	y += drawToolbox(c, y, content)

	return Built{Card: c, Height: y + PadBottom}
}

// drawToolbox lays the tool marks out left to right, wrapping onto a new row
// when it runs out of width, and returns the height it used.
func drawToolbox(c *svgx.Card, top, width float64) float64 {
	step := fleetToolSize + fleetToolGap
	perRow := max(1, int((width+fleetToolGap)/step))

	x, y := PadX, top
	for i, ic := range icons.Toolbox {
		if i > 0 && i%perRow == 0 {
			x = PadX
			y += fleetToolSize + fleetToolRow
		}
		c.Mark(x, y, fleetToolSize, ic.ViewBox, ic.Paths, c.Theme.Muted)
		x += step
	}
	return y + fleetToolSize - top
}
