package cards

import (
	"fmt"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/svgx"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

const (
	codingTotalSize = 34.0
	codingSubSize   = 12.0
	codingChartH    = 58.0
	codingBarGap    = 5.0

	// Each bar is labelled with its own weekday rather than the window being
	// captioned with two dates at the ends. Two letters, because a single
	// initial cannot tell Tuesday from Thursday or Saturday from Sunday.
	codingDaySize = 9.0
	codingDayGap  = 7.0

	// A day with any recorded time keeps a visible sliver rather than
	// collapsing to nothing, so a quiet day reads as quiet and not as missing.
	codingMinBarH = 2.0

	// Weekends are drawn back so the working rhythm is visible at a glance.
	codingWeekendOpacity = 0.45
)

// Coding draws the total, the daily average and the per-day bar chart.
func Coding(t theme.Theme, data model.Coding, days int) Built {
	c := svgx.NewCard(WidthHalf, t)
	content := WidthHalf - 2*PadX

	y := labelWithNote(c, PadTop,
		fmt.Sprintf("$ wakatime --last-%d-days", days), dateRange(data.Days), LabelGap)

	total := time.Duration(data.TotalSeconds) * time.Second
	headline := "—"
	if data.Known() {
		headline = hoursMinutes(total)
	}
	c.Text(PadX, svgx.FirstBaseline(y, lineBox(codingTotalSize), codingTotalSize),
		headline, svgx.Text{
			Size:   codingTotalSize,
			Fill:   t.Strong,
			Weight: 500,
		})
	y += lineBox(codingTotalSize)

	y += 6
	if data.Known() {
		// The average is over the whole window rather than over the days
		// actually worked, which is what "daily average" means on the design.
		var avg time.Duration
		if n := len(data.Days); n > 0 {
			avg = total / time.Duration(n)
		}
		sub := fmt.Sprintf("across %d projects · %s daily average", data.Projects, hoursMinutes(avg))
		c.Text(PadX, svgx.FirstBaseline(y, lineBox(codingSubSize), codingSubSize), sub,
			svgx.Text{Size: codingSubSize, Fill: t.Muted})
	} else {
		c.Text(PadX, svgx.FirstBaseline(y, lineBox(codingSubSize), codingSubSize),
			"no coding time recorded yet", svgx.Text{Size: codingSubSize, Fill: t.Faint})
	}
	y += lineBox(codingSubSize)

	y += 22
	y += drawCodingChart(c, PadX, y, content, data.Days)

	return Built{Card: c, Height: y + PadBottom}
}

// drawCodingChart draws the bars with a weekday under each one, and returns the
// height it used. Bars are scaled so the busiest day fills the chart.
func drawCodingChart(c *svgx.Card, x, top, width float64, days []model.Day) float64 {
	if len(days) == 0 {
		return codingChartH
	}
	peak := 0
	for _, d := range days {
		peak = max(peak, d.Seconds)
	}

	n := float64(len(days))
	barW := (width - codingBarGap*(n-1)) / n
	baseline := top + codingChartH
	labelBase := svgx.FirstBaseline(baseline+codingDayGap, lineBox(codingDaySize), codingDaySize)

	for i, d := range days {
		bx := x + float64(i)*(barW+codingBarGap)

		day, weekend := weekday(d.Date)
		opacity := 1.0
		if weekend {
			opacity = codingWeekendOpacity
		}

		h := 0.0
		if peak > 0 && d.Seconds > 0 {
			h = max(codingMinBarH, codingChartH*float64(d.Seconds)/float64(peak))
		}
		if h > 0 {
			c.RoundRectOpacity(bx, baseline-h, barW, h, 1, c.Theme.Accent, opacity)
		}

		fill := c.Theme.Faint
		if weekend {
			fill = c.Theme.Divider
		}
		c.TextMiddle(bx+barW/2, labelBase, day, svgx.Text{Size: codingDaySize, Fill: fill})
	}

	return codingChartH + codingDayGap + lineBox(codingDaySize)
}

// weekday turns a YYYY-MM-DD date into its two-letter abbreviation, and reports
// whether it falls on a weekend.
func weekday(date string) (string, bool) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", false
	}
	wd := d.Weekday()
	return [...]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}[wd],
		wd == time.Saturday || wd == time.Sunday
}

// dateRange captions the chart with the window it covers, as "Aug 24 – Sep 6".
// The weekday under each bar says which day is which; this only fixes the
// window in the calendar.
func dateRange(days []model.Day) string {
	if len(days) == 0 {
		return ""
	}
	from, err1 := time.Parse("2006-01-02", days[0].Date)
	to, err2 := time.Parse("2006-01-02", days[len(days)-1].Date)
	if err1 != nil || err2 != nil {
		return ""
	}
	if from.Month() == to.Month() {
		return fmt.Sprintf("%s %d – %d", from.Format("Jan"), from.Day(), to.Day())
	}
	return fmt.Sprintf("%s %d – %s %d",
		from.Format("Jan"), from.Day(), to.Format("Jan"), to.Day())
}
