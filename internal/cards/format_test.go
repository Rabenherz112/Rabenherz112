package cards

import (
	"testing"
	"time"
)

func TestComma(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1284, "1,284"},
		{96301, "96,301"},
		{1000000, "1,000,000"},
		{-1234, "-1,234"},
	} {
		if got := comma(tc.in); got != tc.want {
			t.Errorf("comma(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHoursMinutes(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0m"},
		{59 * time.Minute, "59m"},
		{time.Hour, "1h 0m"},
		{222480 * time.Second, "61h 48m"}, // the design's headline figure
		{90*time.Minute + 29*time.Second, "1h 30m"},
	} {
		if got := hoursMinutes(tc.in); got != tc.want {
			t.Errorf("hoursMinutes(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBinarySize(t *testing.T) {
	const tib = int64(1) << 40
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2.0 KiB"},
		{2638827906662, "2.4 TiB"}, // the design's figure
		{15 * tib, "15 TiB"},
	} {
		if got := binarySize(tc.in); got != tc.want {
			t.Errorf("binarySize(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSince(t *testing.T) {
	now := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Minute, "now"},
		{2 * time.Hour, "2h"},
		{25 * time.Hour, "1d"},
		{6 * 24 * time.Hour, "6d"},
		{20 * 24 * time.Hour, "2w"},
		{400 * 24 * time.Hour, "1y"},
	} {
		if got := since(now.Add(-tc.ago), now); got != tc.want {
			t.Errorf("since(%v ago) = %q, want %q", tc.ago, got, tc.want)
		}
	}
}

func TestPct(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{34, "34%"},
		{33.6, "34%"},
		{0.4, "0%"},
		{100, "100%"},
	} {
		if got := pct(tc.in); got != tc.want {
			t.Errorf("pct(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSpaceBetween(t *testing.T) {
	// Three items of 10 across 100: first flush left, last flush right.
	got := spaceBetween(0, 100, []float64{10, 10, 10})
	want := []float64{0, 45, 90}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("spaceBetween[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// A single item sits at the left edge; there is nothing to space it against.
	if got := spaceBetween(5, 100, []float64{10}); len(got) != 1 || got[0] != 5 {
		t.Errorf("spaceBetween of one = %v, want [5]", got)
	}

	// Items wider than the row overlap rather than being pulled backwards.
	got = spaceBetween(0, 10, []float64{10, 10})
	if got[1] < got[0] {
		t.Errorf("spaceBetween overflowed backwards: %v", got)
	}
}
