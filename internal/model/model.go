// Package model holds the data the cards are drawn from, and the snapshot file
// that keeps them drawable when a source is unreachable.
//
// The generator runs on a schedule and commits whatever it produces. If a
// source were allowed to fail the run, one flaky API would either break the
// workflow or blank a card on a live profile page. So every section carries its
// own fetch time, sources write into a snapshot that is committed alongside the
// cards, and a section that fails to refresh simply keeps the values it had.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Snapshot is the whole data set the cards are rendered from.
type Snapshot struct {
	// GeneratedAt is when the cards were last built, successful sources or not.
	GeneratedAt time.Time `json:"generated_at"`

	Languages Languages `json:"languages"`
	Coding    Coding    `json:"coding"`
	Activity  Activity  `json:"activity"`
	AniList   AniList   `json:"anilist"`
	Steam     Steam     `json:"steam"`
	Kavita    Kavita    `json:"kavita"`
}

// Fetched records when a section last refreshed. A section still holding an old
// timestamp is stale data being reused after a failed fetch, which is
// deliberate: a stale card reads better than an empty one.
type Fetched struct {
	FetchedAt time.Time `json:"fetched_at"`
}

// Known reports whether this section has ever been fetched successfully.
//
// A section that has not is drawn as a dash rather than as zeroes or as
// placeholder numbers. Plausible-looking figures that were never measured are
// worse than an obvious blank: there is no way to tell them from real ones, so
// a source that has never worked looks like it is working.
func (f Fetched) Known() bool { return !f.FetchedAt.IsZero() }

// Languages is the share of coding time per language over the last year.
type Languages struct {
	Fetched
	// Items are ordered by share, largest first.
	Items []Language `json:"items"`
}

// A Language is one row of the languages card. The bar colour is not stored:
// it is looked up from the name at render time by package linguist, so the
// palette can be retuned without refetching.
type Language struct {
	Name string  `json:"name"`
	Pct  float64 `json:"pct"`
}

// Coding is the recent coding-time summary.
type Coding struct {
	Fetched
	// TotalSeconds and Projects cover the whole window; Days holds one entry
	// per day, oldest first.
	TotalSeconds int   `json:"total_seconds"`
	Projects     int   `json:"projects"`
	Days         []Day `json:"days"`
}

// A Day is one bar of the coding-time chart.
type Day struct {
	Date    string `json:"date"` // YYYY-MM-DD
	Seconds int    `json:"seconds"`
}

// Activity is the recent public review and pull request history.
type Activity struct {
	Fetched
	Items []Event `json:"items"`
}

// An Event is one line of the activity card.
type Event struct {
	// Verb is one of opened, closed, merged, approved, changes, released or
	// joined. Comments are not shown at all; see sources.convertEvent.
	Verb string `json:"verb"`
	// Num is the pull request or issue number as "#3038", or a release's tag.
	// It is empty for joined, where the repository is the whole story.
	Num  string    `json:"num"`
	Repo string    `json:"repo"`
	URL  string    `json:"url"`
	When time.Time `json:"when"`
}

// AniList is the manga and anime tracking summary.
type AniList struct {
	Fetched
	SeriesTracked int     `json:"series_tracked"`
	ChaptersRead  int     `json:"chapters_read"`
	VolumesRead   int     `json:"volumes_read"`
	MeanScore     float64 `json:"mean_score"`
	Reading       []Media `json:"reading"`
}

// A Media is one series being read, with its cover art.
type Media struct {
	Title string `json:"title"`
	Meta  string `json:"meta"` // "ch. 187"
	// Cover is the cover image, already fetched and inlined, since a card
	// cannot load one at view time. Empty when the art could not be had.
	Cover     []byte `json:"cover,omitempty"`
	CoverMIME string `json:"cover_mime,omitempty"`
}

// Steam is the recently played summary.
type Steam struct {
	Fetched
	OwnedGames int    `json:"owned_games"`
	Recent     []Game `json:"recent"`
}

// A Game is one row of the Steam card.
type Game struct {
	Name           string `json:"name"`
	TwoWeekMinutes int    `json:"two_week_minutes"`
	// Icon is the game's store icon, already fetched and inlined, since a card
	// cannot load one at view time. Empty when it could not be had.
	Icon     []byte `json:"icon,omitempty"`
	IconMIME string `json:"icon_mime,omitempty"`
}

// Kavita is the self-hosted manga library summary.
type Kavita struct {
	Fetched
	Series int   `json:"series"`
	Bytes  int64 `json:"bytes"`
	Genres int   `json:"genres"`
}

// ErrCorrupt reports a snapshot file that exists but could not be parsed.
//
// It is worth telling apart from an unreadable file. The snapshot is generated
// output that every run rewrites, so a corrupt one is recoverable: start empty,
// refetch, and the next commit is clean again. An unreadable one is a problem
// with the machine and there is nothing sensible to do about it here.
var ErrCorrupt = errors.New("snapshot could not be parsed")

// Load reads a snapshot from disk. A missing file is not an error: the first
// run has nothing to fall back on and every section starts empty.
//
// A file that exists but does not parse returns an empty snapshot together
// with an error wrapping ErrCorrupt, so a caller can choose to carry on.
func Load(path string) (*Snapshot, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Snapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return &Snapshot{}, fmt.Errorf("%w: %s: %v", ErrCorrupt, path, err)
	}
	return &s, nil
}

// Save writes the snapshot back, indented so a diff shows what actually moved.
func Save(path string, s *Snapshot) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
