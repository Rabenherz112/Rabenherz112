package cards

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// comma formats a whole number with thousands separators, the way the design
// writes 96,301.
func comma(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// hoursMinutes formats a duration as "61h 48m", dropping the hours when there
// are none.
func hoursMinutes(d time.Duration) string {
	m := int(d.Round(time.Minute) / time.Minute)
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %dm", m/60, m%60)
}

// binarySize formats a byte count in binary units, as the design's "2.4 TiB".
func binarySize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 4; v /= unit {
		div *= unit
		exp++
	}
	v := float64(n) / float64(div)
	units := [...]string{"KiB", "MiB", "GiB", "TiB", "PiB"}

	// One decimal below ten, none above, so the figure stays about as wide
	// whatever the size.
	if v < 10 {
		return fmt.Sprintf("%.1f %s", v, units[exp])
	}
	return fmt.Sprintf("%.0f %s", math.Round(v), units[exp])
}

// since formats an age the way a git log does: "2h", "1d", "3w". Anything under
// an hour reads as "now", since the card is only rebuilt daily and a minute
// count would be wrong by the time anyone saw it.
func since(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Hour:
		return "now"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h"
	case d < 7*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/24)) + "d"
	case d < 365*24*time.Hour:
		return strconv.Itoa(int(d.Hours()/(24*7))) + "w"
	default:
		return strconv.Itoa(int(d.Hours()/(24*365))) + "y"
	}
}

// pct formats a share as the design writes it: whole percent, no decimals.
func pct(v float64) string {
	return strconv.Itoa(int(math.Round(v))) + "%"
}
