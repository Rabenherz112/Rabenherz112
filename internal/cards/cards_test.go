package cards

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
	"github.com/Rabenherz112/Rabenherz112/internal/wordmark"
)

// hostileSnapshot is what the sources might hand over on a bad day: text with
// XML metacharacters in it, values long enough to need truncating, an empty
// section, and a zero where a divisor is expected.
func hostileSnapshot() *model.Snapshot {
	now := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	return &model.Snapshot{
		Languages: model.Languages{Items: []model.Language{
			{Name: "C++ & <friends>", Pct: 61.5},
			{Name: "AnAbsurdlyLongLanguageNameThatCannotFit", Pct: 38.5},
		}},
		Coding: model.Coding{
			TotalSeconds: 0, // a week with nothing recorded
			Projects:     0,
			Days: []model.Day{
				{Date: "2026-09-05", Seconds: 0},
				{Date: "2026-09-06", Seconds: 0},
			},
		},
		Activity: model.Activity{Items: []model.Event{
			{Verb: "approved", Num: "#1", Repo: "owner/repo", When: now.Add(-2 * time.Hour)},
			{Verb: "surprise", Num: "#2", Repo: strings.Repeat("very-long-org/", 12) + "repo", When: now.Add(-72 * time.Hour)},
		}},
		AniList: model.AniList{
			SeriesTracked: 1284,
			ChaptersRead:  96301,
			MeanScore:     7.8,
			Reading: []model.Media{
				{Title: `Fullmetal Alchemist & "Brotherhood" <ova>`, Meta: "ch. 1"},
				{Title: "Short", Meta: "ch. 2"},
			},
		},
		Steam: model.Steam{
			OwnedGames: 341,
			Recent: []model.Game{
				{Name: "A Game With A Very Long Name Indeed", TwoWeekMinutes: 0},
			},
		},
		Kavita: model.Kavita{Series: 1284, Bytes: 2638827906662, Genres: 41},
	}
}

// assertWellFormed fails when the SVG will not parse, which is how an
// unescaped ampersand from an API would show up.
func assertWellFormed(t *testing.T, name string, svg []byte) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(string(svg)))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("%s is not well-formed XML: %v", name, err)
		}
	}
}

func TestAllCardsRenderWellFormed(t *testing.T) {
	snap := hostileSnapshot()
	now := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	mark, err := wordmark.Read("../../assets/wordmark-dark.png")
	if err != nil {
		t.Fatalf("load wordmark: %v", err)
	}

	for _, th := range theme.Both {
		langs, coding := RenderPair(Languages(th, snap.Languages), Coding(th, snap.Coding, 14))
		steam, kavita := RenderPair(Steam(th, snap.Steam), Kavita(th, snap.Kavita))

		for name, svg := range map[string][]byte{
			"header":    Header(th, mark).Render(),
			"fleet":     Fleet(th).Render(),
			"activity":  Activity(th, snap.Activity, "updated daily", now).Render(),
			"anilist":   AniList(th, snap.AniList, "anilist.co/user/x").Render(),
			"languages": langs,
			"wakatime":  coding,
			"steam":     steam,
			"kavita":    kavita,
		} {
			assertWellFormed(t, name+"-"+th.Name, svg)
		}
	}
}

// TestEmptySectionsRender covers the first run, and any run where every source
// failed before the snapshot had anything in it.
func TestEmptySectionsRender(t *testing.T) {
	var snap model.Snapshot
	now := time.Now()
	th := theme.Dark

	langs, coding := RenderPair(Languages(th, snap.Languages), Coding(th, snap.Coding, 14))
	steam, kavita := RenderPair(Steam(th, snap.Steam), Kavita(th, snap.Kavita))

	for name, svg := range map[string][]byte{
		"activity":  Activity(th, snap.Activity, "updated daily", now).Render(),
		"anilist":   AniList(th, snap.AniList, "note").Render(),
		"languages": langs,
		"wakatime":  coding,
		"steam":     steam,
		"kavita":    kavita,
	} {
		assertWellFormed(t, name, svg)
	}
}

// TestRenderPairSquaresOff checks the two halves come out the same height.
// Markdown scales each by its own aspect ratio, so a mismatch shows up as a
// ragged bottom edge between them.
func TestRenderPairSquaresOff(t *testing.T) {
	snap := hostileSnapshot()
	th := theme.Dark

	a := Languages(th, snap.Languages)
	b := Coding(th, snap.Coding, 14)
	if a.Height == b.Height {
		t.Skip("the two cards happen to be the same height already")
	}

	left, right := RenderPair(a, b)
	if h1, h2 := svgHeight(t, left), svgHeight(t, right); h1 != h2 {
		t.Errorf("paired cards are %s and %s tall, want equal", h1, h2)
	}
}

// TestRenderPairTilesTheRow checks the two halves add up to a full-width card.
//
// The README places them at 50% each with no space between, so anything other
// than exactly half the row leaves the pair narrower than every full-width
// card on the page, or wide enough that the second one wraps onto its own
// line. The gap between the boxes has to be inside the images.
func TestRenderPairTilesTheRow(t *testing.T) {
	snap := hostileSnapshot()
	th := theme.Dark

	left, right := RenderPair(Steam(th, snap.Steam), Kavita(th, snap.Kavita))

	want := num(WidthFull / 2)
	for name, svg := range map[string][]byte{"left": left, "right": right} {
		if got := svgWidth(t, svg); got != want {
			t.Errorf("%s half is %s wide, want %s so two tile a %g row",
				name, got, want, WidthFull)
		}
	}

	// The right half carries its gutter on the left, so its frame is pushed
	// clear of the left half's rather than sitting flush against it.
	if !strings.Contains(string(right), `transform="translate(`+num(PairGutter)+` 0)"`) {
		t.Errorf("the right half does not offset its contents by the gutter:\n%s",
			string(right)[:min(len(right), 400)])
	}
}

// num formats a coordinate the way the rendered SVG does.
func num(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0"), ".")
}

// svgWidth pulls the width attribute off a rendered card.
func svgWidth(t *testing.T, svg []byte) string {
	t.Helper()
	return svgAttr(t, svg, "width")
}

// svgHeight pulls the height attribute off a rendered card.
func svgHeight(t *testing.T, svg []byte) string {
	t.Helper()
	return svgAttr(t, svg, "height")
}

// svgAttr pulls the first occurrence of an attribute off a rendered card,
// which for width and height is the one on the <svg> element itself.
func svgAttr(t *testing.T, svg []byte, name string) string {
	t.Helper()
	s := string(svg)
	key := name + `="`
	i := strings.Index(s, key)
	if i < 0 {
		t.Fatalf("no %s attribute on the card", name)
	}
	rest := s[i+len(key):]
	return rest[:strings.IndexByte(rest, '"')]
}
