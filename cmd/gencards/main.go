// Command gencards builds the SVG cards that make up the profile README.
//
// It fetches from every configured source, writes the results to a snapshot
// file, and renders each card twice, once per palette. Sources are allowed to
// fail: this runs unattended on a schedule and commits what it produces, so one
// unreachable API must not blank a card on a live page. A failing source is
// logged and its section of the snapshot keeps the values it already had.
//
// Configuration is entirely by environment variable, since that is how the
// workflow supplies its secrets. Every source is optional; one with no
// credentials is skipped and its card renders from the snapshot.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/cards"
	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/sources"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
	"github.com/Rabenherz112/Rabenherz112/internal/wordmark"
)

// How much of each source the cards have room for. These are layout facts, so
// they live with the program rather than in configuration.
const (
	activityLines  = 10
	topLanguages   = 5
	chartDays      = 14
	readingEntries = 3
	steamEntries   = 3
)

// cadence is how often the workflow runs, printed on the cards that claim to be
// kept up to date. Change it together with the schedule in the workflow.
const cadence = "daily"

// fetchTimeout bounds the whole fetch phase, so a hanging source cannot keep
// the workflow running until Actions kills it.
const fetchTimeout = 2 * time.Minute

// config is everything a run needs. It is assembled from the environment and
// the flags by configFromEnv, and built directly by tests.
type config struct {
	OutDir       string
	SnapshotPath string
	AssetsDir    string
	ReadmePath   string // empty to leave the README alone
	Offline      bool

	GitHub  sources.GitHubConfig
	Wakapi  sources.WakapiConfig
	AniList sources.AniListConfig
	Steam   sources.SteamConfig
	Kavita  sources.KavitaConfig
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gencards: ")

	cfg := configFromEnv()
	flag.StringVar(&cfg.OutDir, "out", cfg.OutDir, "directory to write the card SVGs into")
	flag.StringVar(&cfg.SnapshotPath, "snapshot", cfg.SnapshotPath, "snapshot file to read and update")
	flag.StringVar(&cfg.AssetsDir, "assets", cfg.AssetsDir, "directory holding the source artwork")
	flag.StringVar(&cfg.ReadmePath, "readme", cfg.ReadmePath, "README to stamp with the build date (empty to skip)")
	flag.BoolVar(&cfg.Offline, "offline", cfg.Offline, "skip every fetch and render from the snapshot as it stands")
	flag.Parse()

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

// env reads a variable, falling back to def when it is unset or empty.
func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// configFromEnv assembles the run's configuration.
//
// Public identities carry defaults, because they are printed on the cards
// anyway. The self-hosted addresses do not: Wakapi and Kavita run on the
// profile owner's own machines, and their URLs arrive as secrets rather than
// being written down here. A source with no address configured is skipped.
func configFromEnv() config {
	ghUser := env("PROFILE_GITHUB_USER", "Rabenherz112")

	return config{
		OutDir:       "cards",
		SnapshotPath: filepath.Join("data", "snapshot.json"),
		AssetsDir:    "assets",
		ReadmePath:   "README.md",

		GitHub: sources.GitHubConfig{
			User:  ghUser,
			Token: os.Getenv("GITHUB_TOKEN"),
			// The profile repository commits to itself on every run, which
			// would otherwise crowd out everything worth showing.
			Ignore: []string{ghUser + "/" + ghUser},
			Limit:  activityLines,
		},
		Wakapi: sources.WakapiConfig{
			BaseURL:      os.Getenv("WAKAPI_URL"),
			APIKey:       os.Getenv("WAKAPI_API_KEY"),
			TopLanguages: topLanguages,
			ChartDays:    chartDays,
		},
		AniList: sources.AniListConfig{
			User:    env("ANILIST_USER", "Rabenherz"),
			Reading: readingEntries,
		},
		Steam: sources.SteamConfig{
			APIKey:    os.Getenv("STEAM_API_KEY"),
			SteamID:   os.Getenv("STEAM_ID"),
			VanityURL: env("STEAM_VANITY", "rabenherz"),
			Recent:    steamEntries,
		},
		Kavita: sources.KavitaConfig{
			BaseURL: os.Getenv("KAVITA_URL"),
			APIKey:  os.Getenv("KAVITA_API_KEY"),
		},
	}
}

func run(cfg config) error {
	snap, err := model.Load(cfg.SnapshotPath)
	if errors.Is(err, model.ErrCorrupt) {
		// The snapshot is generated output, not something anyone hand-writes,
		// and a merge across two runs of this generator will conflict on it
		// every time. Refusing to run would leave the workflow red until
		// somebody hand-repaired a file the next run overwrites anyway. So say
		// so loudly and start from nothing: the sources refetch, and the run
		// ends with a clean snapshot and clean cards.
		log.Printf("WARNING: %v", err)
		log.Print("WARNING: starting from an empty snapshot, so any source that " +
			"fails this run has no previous values to fall back on")
	} else if err != nil {
		return err
	}

	now := time.Now().UTC().Truncate(time.Second)
	if cfg.Offline {
		log.Print("offline: rendering from the existing snapshot")
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		refresh(ctx, cfg, snap, now)
	}

	snap.GeneratedAt = now
	if err := model.Save(cfg.SnapshotPath, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	if err := render(cfg, snap, now); err != nil {
		return err
	}
	if cfg.ReadmePath != "" {
		if err := stampReadme(cfg.ReadmePath, now); err != nil {
			return fmt.Errorf("stamp readme: %w", err)
		}
	}
	return nil
}

// refresh updates every section it has credentials for, in parallel.
//
// Nothing here returns an error. A source that fails logs a warning and leaves
// its section of the snapshot untouched, which is the whole point: a page with
// yesterday's numbers on one card is a far better outcome than a failed
// workflow or a blank card, and self-hosted services do go down.
func refresh(ctx context.Context, cfg config, snap *model.Snapshot, now time.Time) {
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	// fetch runs one source, unless it has nothing configured to reach. The
	// work runs concurrently; the mutex guards only the write back into the
	// shared snapshot.
	fetch := func(name string, configured bool, do func() error) {
		if !configured {
			log.Printf("%s: not configured, keeping the last snapshot", name)
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := do(); err != nil {
				log.Printf("%s: %v (keeping the last snapshot)", name, err)
			}
		}()
	}

	// GitHub works unauthenticated too, at a much lower rate limit. In the
	// workflow the token is always present.
	fetch("github", true, func() error {
		got, err := sources.FetchGitHub(ctx, cfg.GitHub)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.Activity = got
		return nil
	})

	wakapiReady := cfg.Wakapi.BaseURL != "" && cfg.Wakapi.APIKey != ""
	fetch("wakapi languages", wakapiReady, func() error {
		got, err := sources.FetchLanguages(ctx, cfg.Wakapi)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.Languages = got
		return nil
	})
	fetch("wakapi coding time", wakapiReady, func() error {
		got, err := sources.FetchCoding(ctx, cfg.Wakapi, now)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.Coding = got
		return nil
	})

	// AniList is a public GraphQL API; there is nothing to configure.
	fetch("anilist", true, func() error {
		got, err := sources.FetchAniList(ctx, cfg.AniList)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.AniList = got
		return nil
	})

	fetch("steam", cfg.Steam.APIKey != "", func() error {
		got, err := sources.FetchSteam(ctx, cfg.Steam)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.Steam = got
		return nil
	})

	fetch("kavita", cfg.Kavita.BaseURL != "" && cfg.Kavita.APIKey != "", func() error {
		got, err := sources.FetchKavita(ctx, cfg.Kavita)
		if err != nil {
			return err
		}
		mu.Lock()
		defer mu.Unlock()
		snap.Kavita = got
		return nil
	})

	wg.Wait()
	warnIfStale(snap, now)
}

// staleAfter is how long a section may go without a successful refresh before
// the run says so. Failing soft keeps a card looking fine on the page, so the
// workflow log is the only place a long outage would otherwise surface.
const staleAfter = 7 * 24 * time.Hour

// warnIfStale reports sections that have not refreshed in a long time. It is
// the counterpart to failing soft: keeping old values is right, doing it
// silently forever is not.
func warnIfStale(snap *model.Snapshot, now time.Time) {
	for _, s := range []struct {
		name string
		at   time.Time
	}{
		{"activity", snap.Activity.FetchedAt},
		{"languages", snap.Languages.FetchedAt},
		{"coding time", snap.Coding.FetchedAt},
		{"anilist", snap.AniList.FetchedAt},
		{"steam", snap.Steam.FetchedAt},
		{"kavita", snap.Kavita.FetchedAt},
	} {
		if s.at.IsZero() {
			continue // never fetched; the "not configured" line already said so
		}
		if age := now.Sub(s.at); age > staleAfter {
			log.Printf("WARNING: %s last refreshed %d days ago, so its card is showing stale numbers",
				s.name, int(age.Hours()/24))
		}
	}
}

// render draws every card in both palettes and writes them out.
func render(cfg config, snap *model.Snapshot, now time.Time) error {
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return err
	}

	anilistNote := "anilist.co/user/" + cfg.AniList.User
	activityNote := "updated " + cadence

	for _, t := range theme.Both {
		// Already re-encoded and committed by tools/genwordmark, so that no
		// image compressor runs during a build. See package wordmark.
		mark, err := wordmark.Read(filepath.Join(cfg.AssetsDir, "wordmark-"+t.Name+".png"))
		if err != nil {
			return fmt.Errorf("read wordmark: %w", err)
		}

		langs, coding := cards.RenderPair(
			cards.Languages(t, snap.Languages),
			cards.Coding(t, snap.Coding, chartDays),
		)
		steam, kavita := cards.RenderPair(
			cards.Steam(t, snap.Steam),
			cards.Kavita(t, snap.Kavita),
		)

		files := map[string][]byte{
			"header":    cards.Header(t, mark).Render(),
			"fleet":     cards.Fleet(t).Render(),
			"activity":  cards.Activity(t, snap.Activity, activityNote, now).Render(),
			"anilist":   cards.AniList(t, snap.AniList, anilistNote).Render(),
			"languages": langs,
			"wakatime":  coding,
			"steam":     steam,
			"kavita":    kavita,
		}
		// The link row is the only clickable graphic on the page, so each tile
		// has to be its own image for a link to wrap it.
		for _, l := range cards.Links {
			files["link-"+l.Name] = cards.Tile(t, l)
		}

		for name, svg := range files {
			path := filepath.Join(cfg.OutDir, fmt.Sprintf("%s-%s.svg", name, t.Name))
			if err := os.WriteFile(path, svg, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	}
	return nil
}

// stampMarker wraps the one generated line in the README. Everything else in
// that file is hand-written and is never touched.
var stampMarker = regexp.MustCompile(`(?s)(<!-- generated:start -->).*?(<!-- generated:end -->)`)

// stampVisible controls whether the build-date line shows on the page.
//
// With it false the line is still written and still kept current, just inside
// an HTML comment, so putting it back on the page is this one constant.
// Commenting the block out in the README instead does not work, twice over:
// the markers are comments themselves and HTML comments do not nest, and every
// run rewrites whatever is between them anyway.
const stampVisible = false

// stampReadme rewrites the build-date line between the generated markers.
func stampReadme(path string, now time.Time) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !stampMarker.Match(b) {
		return fmt.Errorf("no <!-- generated:start --> block in %s", path)
	}
	line := fmt.Sprintf("<p align=\"center\"><sub>generated %s · rebuilt %s by <code>.github/workflows</code></sub></p>",
		now.Format("2006-01-02"), cadence)
	if !stampVisible {
		line = "<!-- " + line + " -->"
	}
	out := stampMarker.ReplaceAll(b, []byte("${1}\n"+line+"\n${2}"))
	return os.WriteFile(path, out, 0o644)
}
