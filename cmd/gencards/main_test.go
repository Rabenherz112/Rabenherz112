// End-to-end tests for the generator, run against stub servers standing in for
// the real APIs.
//
// The case these exist for is a self-hosted service being down. Wakapi and
// Kavita run on the profile owner's own hardware; the box gets rebooted, the
// certificate expires, the tunnel drops. When that happens the run has to
// succeed anyway, keep the numbers it had, and still write every card. These
// tests hold that behaviour in place.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/cards"
	"github.com/Rabenherz112/Rabenherz112/internal/model"
	"github.com/Rabenherz112/Rabenherz112/internal/sources"
	"github.com/Rabenherz112/Rabenherz112/internal/theme"
)

// fakeAPIs is a set of stub servers, one per source, that a test can take down
// individually.
type fakeAPIs struct {
	github  *httptest.Server
	wakapi  *httptest.Server
	anilist *httptest.Server
	steam   *httptest.Server
	kavita  *httptest.Server
}

func reply(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func startFakeAPIs(t *testing.T) *fakeAPIs {
	t.Helper()

	serve := func(routes map[string]http.HandlerFunc) *httptest.Server {
		mux := http.NewServeMux()
		for p, h := range routes {
			mux.HandleFunc(p, h)
		}
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv
	}

	f := &fakeAPIs{
		github: serve(map[string]http.HandlerFunc{
			"/users/tester/events/public": reply(`[
				{"type":"PullRequestReviewEvent","created_at":"2026-09-06T12:00:00Z",
				 "repo":{"name":"awesome-selfhosted/awesome-selfhosted-data"},
				 "payload":{"review":{"state":"approved","html_url":"https://gh/1"},
				            "pull_request":{"number":3038}}}]`),
		}),
		wakapi: serve(map[string]http.HandlerFunc{
			"/api/compat/wakatime/v1/users/current/stats/last_year": reply(
				`{"data":{"languages":[{"name":"Go","percent":58.5},{"name":"PowerShell","percent":41.5}]}}`),
			"/api/compat/wakatime/v1/users/current/summaries": reply(
				`{"data":[{"range":{"date":"2026-09-06"},"grand_total":{"total_seconds":7200},
				           "projects":[{"name":"profile","total_seconds":7200}]}]}`),
		}),
		anilist: serve(map[string]http.HandlerFunc{
			"/": reply(`{"data":{
				"User":{"statistics":{"manga":{"count":1284,"chaptersRead":96301,"volumesRead":4912,"meanScore":78}}},
				"MediaListCollection":{"lists":[{"entries":[
					{"progress":187,"updatedAt":300,
					 "media":{"title":{"romaji":"Vinland Saga","english":"Vinland Saga"},
					          "coverImage":{"medium":""}}}]}]}}}`),
		}),
		steam: serve(map[string]http.HandlerFunc{
			"/IPlayerService/GetRecentlyPlayedGames/v1/": reply(
				`{"response":{"games":[{"name":"Hades II","playtime_2weeks":744}]}}`),
			"/IPlayerService/GetOwnedGames/v1/": reply(`{"response":{"game_count":341}}`),
		}),
		kavita: serve(map[string]http.HandlerFunc{
			"/api/Plugin/authenticate": reply(`{"token":"jwt"}`),
			"/api/Stats/server/stats": reply(
				`{"seriesCount":1284,"totalSize":2638827906662,"totalGenres":41}`),
		}),
	}
	return f
}

// workspace lays out a temporary repository: the artwork the header card needs,
// and a README with the marker block the run stamps.
func workspace(t *testing.T, f *fakeAPIs) config {
	t.Helper()
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The header card embeds the prepared wordmarks, which are committed rather
	// than made at build time. See package wordmark.
	for _, name := range []string{"wordmark-dark.png", "wordmark-light.png"} {
		art, err := os.ReadFile(filepath.Join("..", "..", "assets", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "assets", name), art, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme,
		[]byte("intro\n<!-- generated:start -->\nold\n<!-- generated:end -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return config{
		OutDir:       filepath.Join(dir, "cards"),
		SnapshotPath: filepath.Join(dir, "data", "snapshot.json"),
		AssetsDir:    filepath.Join(dir, "assets"),
		ReadmePath:   readme,

		GitHub:  sources.GitHubConfig{BaseURL: f.github.URL, User: "tester", Limit: 10},
		Wakapi:  sources.WakapiConfig{BaseURL: f.wakapi.URL, APIKey: "k", TopLanguages: 5, ChartDays: 1},
		AniList: sources.AniListConfig{BaseURL: f.anilist.URL, User: "tester", Reading: 3},
		Steam:   sources.SteamConfig{BaseURL: f.steam.URL, APIKey: "k", SteamID: "1", Recent: 3},
		Kavita:  sources.KavitaConfig{BaseURL: f.kavita.URL, APIKey: "k", Host: "manga.example.com"},
	}
}

func loadSnapshot(t *testing.T, path string) *model.Snapshot {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var s model.Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("parse snapshot: %v", err)
	}
	return &s
}

// captureLog collects what the run logged, so a test can assert on the warning
// a failing source is supposed to leave behind.
func captureLog(t *testing.T) *strings.Builder {
	t.Helper()
	var buf strings.Builder
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	return &buf
}

// panelCount is how many full cards the generator draws, as opposed to the
// link tiles, which are counted from the list itself so this stays right when
// a link is added or dropped.
const panelCount = 8

var cardCount = (panelCount + len(cards.Links)) * len(theme.Both)

func TestRunEndToEnd(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)

	if err := run(cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	entries, err := os.ReadDir(cfg.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != cardCount {
		t.Errorf("wrote %d cards, want %d", len(entries), cardCount)
	}

	snap := loadSnapshot(t, cfg.SnapshotPath)
	if len(snap.Activity.Items) != 1 || snap.Activity.Items[0].Num != "#3038" {
		t.Errorf("activity = %+v", snap.Activity.Items)
	}
	if len(snap.Languages.Items) != 2 || snap.Languages.Items[0].Name != "Go" {
		t.Errorf("languages = %+v", snap.Languages.Items)
	}
	if snap.Coding.TotalSeconds != 7200 {
		t.Errorf("coding = %+v", snap.Coding)
	}
	if snap.AniList.ChaptersRead != 96301 {
		t.Errorf("anilist = %+v", snap.AniList)
	}
	if snap.Steam.OwnedGames != 341 {
		t.Errorf("steam = %+v", snap.Steam)
	}
	if snap.Kavita.Series != 1284 {
		t.Errorf("kavita = %+v", snap.Kavita)
	}

	// The README's marked block is stamped, and nothing else in it is touched.
	readme, err := os.ReadFile(cfg.ReadmePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(readme), "intro\n") {
		t.Error("the run rewrote more of the README than the marked block")
	}
	if strings.Contains(string(readme), "old") {
		t.Error("the marked block was not replaced")
	}
	if !strings.Contains(string(readme), "generated "+time.Now().UTC().Format("2006-01-02")) {
		t.Errorf("README was not stamped with today's date:\n%s", readme)
	}
}

// TestSelfHostedOutageKeepsTheCards is the scenario this whole design exists
// for. Kavita is on hardware that is not always up. When it is down the run
// must still succeed, the card must still be written, and it must still show
// the numbers from the last time the fetch worked.
func TestSelfHostedOutageKeepsTheCards(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)

	// A good run first, so there is something to fall back to.
	if err := run(cfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	before := loadSnapshot(t, cfg.SnapshotPath)
	goodCard, err := os.ReadFile(filepath.Join(cfg.OutDir, "kavita-dark.svg"))
	if err != nil {
		t.Fatal(err)
	}

	// The instance goes away: not an error response, but nothing listening at
	// all, which is what a rebooted box or a dropped tunnel looks like.
	f.kavita.Close()

	// Something the user did change in the meantime, to prove the run is not
	// simply frozen: a new review landed.
	f.github.Close()
	f.github = httptest.NewServer(http.HandlerFunc(reply(`[
		{"type":"PullRequestReviewEvent","created_at":"2026-09-07T12:00:00Z",
		 "repo":{"name":"louislam/uptime-kuma"},
		 "payload":{"review":{"state":"approved","html_url":"https://gh/2"},
		            "pull_request":{"number":999}}}]`)))
	t.Cleanup(f.github.Close)
	cfg.GitHub.BaseURL = f.github.URL

	logs := captureLog(t)
	if err := run(cfg); err != nil {
		t.Fatalf("a self-hosted service being down must not fail the run: %v", err)
	}

	after := loadSnapshot(t, cfg.SnapshotPath)

	// Kavita's numbers survive the outage untouched, timestamp included, so
	// the age of the data is still visible.
	if after.Kavita != before.Kavita {
		t.Errorf("kavita values changed during an outage:\nbefore %+v\nafter  %+v", before.Kavita, after.Kavita)
	}
	if after.Kavita.Series == 0 {
		t.Error("kavita was zeroed by the outage")
	}

	// The card is still written, and is byte-identical to the one from the
	// good run.
	staleCard, err := os.ReadFile(filepath.Join(cfg.OutDir, "kavita-dark.svg"))
	if err != nil {
		t.Fatalf("the kavita card was not written during the outage: %v", err)
	}
	if string(staleCard) != string(goodCard) {
		t.Error("the kavita card changed even though its data did not")
	}

	// Everything else still refreshed.
	if len(after.Activity.Items) != 1 || after.Activity.Items[0].Num != "#999" {
		t.Errorf("the other sources did not refresh: %+v", after.Activity.Items)
	}

	// And the failure is not silent.
	if !strings.Contains(logs.String(), "kavita:") ||
		!strings.Contains(logs.String(), "keeping the last snapshot") {
		t.Errorf("the outage was not logged:\n%s", logs)
	}
}

// TestUnconfiguredSourcesAreSkipped covers the first-time setup: secrets not
// added yet, so those sources are skipped rather than attempted and failed.
func TestUnconfiguredSourcesAreSkipped(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)

	// Seed a snapshot so there is something for the skipped cards to draw.
	if err := run(cfg); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	seeded := loadSnapshot(t, cfg.SnapshotPath)

	// Now drop the self-hosted credentials, as they would be before the
	// secrets are added to the repository.
	cfg.Wakapi.APIKey = ""
	cfg.Wakapi.BaseURL = ""
	cfg.Kavita.APIKey = ""
	cfg.Kavita.BaseURL = ""
	cfg.Steam.APIKey = ""

	logs := captureLog(t)
	if err := run(cfg); err != nil {
		t.Fatalf("run with no credentials: %v", err)
	}

	after := loadSnapshot(t, cfg.SnapshotPath)
	if after.Kavita != seeded.Kavita || after.Steam.OwnedGames != seeded.Steam.OwnedGames {
		t.Error("skipped sources should leave their snapshot section alone")
	}
	for _, want := range []string{"wakapi languages: not configured", "kavita: not configured", "steam: not configured"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("missing log line %q in:\n%s", want, logs)
		}
	}

	entries, err := os.ReadDir(cfg.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != cardCount {
		t.Errorf("wrote %d cards, want all %d even with sources skipped", len(entries), cardCount)
	}
}

// TestFirstRunWithNoSnapshot covers a fresh checkout where every source is
// unreachable: no data at all. The cards still have to come out, empty but
// well-formed, rather than the run failing.
func TestFirstRunWithNoSnapshot(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)
	f.github.Close()
	f.wakapi.Close()
	f.anilist.Close()
	f.steam.Close()
	f.kavita.Close()

	if err := run(cfg); err != nil {
		t.Fatalf("run with nothing reachable: %v", err)
	}
	entries, err := os.ReadDir(cfg.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != cardCount {
		t.Errorf("wrote %d cards, want %d", len(entries), cardCount)
	}
}

// TestStaleDataIsWarnedAbout checks the counterpart to failing soft: a section
// that has not refreshed for a long time is called out, so an outage that
// lasts is not hidden behind a card that still looks fine.
func TestStaleDataIsWarnedAbout(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	snap := &model.Snapshot{}
	snap.Kavita.FetchedAt = now.Add(-30 * 24 * time.Hour)
	snap.Activity.FetchedAt = now.Add(-time.Hour)

	logs := captureLog(t)
	warnIfStale(snap, now)

	if !strings.Contains(logs.String(), "kavita last refreshed 30 days ago") {
		t.Errorf("stale kavita was not reported:\n%s", logs)
	}
	if strings.Contains(logs.String(), "activity") {
		t.Errorf("fresh activity should not be reported:\n%s", logs)
	}
	// A section that has never been fetched is covered by the "not configured"
	// line and should not also be called stale.
	if strings.Contains(logs.String(), "steam") {
		t.Errorf("a never-fetched section should not be called stale:\n%s", logs)
	}
}

func TestStampReadmeRequiresMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(path, []byte("no markers here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Losing the stamp silently would leave a wrong date on the page forever,
	// so a missing marker is an error rather than a no-op.
	if err := stampReadme(path, time.Now()); err == nil {
		t.Fatal("want an error when the marker block is missing, got nil")
	}
}

func TestMain(m *testing.M) {
	// These tests assert on log output, so keep it out of the test report.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// TestCorruptSnapshotIsRecoverable covers a merge conflict landing in
// data/snapshot.json.
//
// That file is generated output which every run rewrites, so two branches will
// always disagree about it and a merge will always conflict. Refusing to run
// would leave the scheduled workflow failing until somebody hand-repaired a
// file the next run overwrites anyway, so the run says so loudly, starts from
// nothing, and ends with a clean snapshot and a full set of cards.
func TestCorruptSnapshotIsRecoverable(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)

	if err := os.MkdirAll(filepath.Dir(cfg.SnapshotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	conflicted := "{\n<<<<<<< HEAD\n  \"generated_at\": \"2026-09-06T00:00:00Z\"\n" +
		"=======\n  \"generated_at\": \"2026-09-07T00:00:00Z\"\n>>>>>>> other\n}\n"
	if err := os.WriteFile(cfg.SnapshotPath, []byte(conflicted), 0o644); err != nil {
		t.Fatal(err)
	}

	logs := captureLog(t)
	if err := run(cfg); err != nil {
		t.Fatalf("a conflicted snapshot must not fail the run: %v", err)
	}

	if !strings.Contains(logs.String(), "could not be parsed") {
		t.Errorf("the corruption was not reported:\n%s", logs)
	}

	// The run rewrote it, so the next one starts from something valid.
	snap := loadSnapshot(t, cfg.SnapshotPath)
	if len(snap.Activity.Items) == 0 {
		t.Error("the sources did not refill the snapshot")
	}

	entries, err := os.ReadDir(cfg.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != cardCount {
		t.Errorf("wrote %d cards, want all %d despite the bad snapshot", len(entries), cardCount)
	}
}

// TestUnreadableSnapshotIsFatal is the other half: a file that cannot be read
// at all is a problem with the machine, not recoverable output, and the run
// should stop rather than quietly discard whatever was there.
func TestUnreadableSnapshotIsFatal(t *testing.T) {
	f := startFakeAPIs(t)
	cfg := workspace(t, f)
	// A directory where a file belongs fails the read on every platform.
	if err := os.MkdirAll(cfg.SnapshotPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := run(cfg); err == nil {
		t.Fatal("want an error when the snapshot cannot be read, got nil")
	}
}
