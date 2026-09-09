// Tests for the source layer, run against stub HTTP servers standing in for
// the real APIs. The response bodies below are trimmed copies of what each
// service actually returns, so a change in field names or nesting shows up
// here rather than as an empty card three weeks later.
//
// Every source gets two kinds of case: the shape it expects, and the ways it
// can go wrong. The failure cases matter as much as the happy path, because a
// source returning an error is how a card keeps its previous values instead of
// rendering blank.
package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stub starts a test server that routes on the request path, and returns its
// URL. Handlers get the raw request so a test can assert on query parameters
// and headers.
func stub(t *testing.T, routes map[string]http.HandlerFunc) string {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range routes {
		mux.HandleFunc(path, h)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

// replyJSON replies with a fixed body.
func replyJSON(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// fails replies with an error status, the way an overloaded or misconfigured
// service does.
func fails(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"nope"}`, code)
	}
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return c
}

// ---------------------------------------------------------------- GitHub ---

const githubEvents = `[
  {"type":"PullRequestReviewEvent","created_at":"2026-09-06T12:00:00Z",
   "repo":{"name":"awesome-selfhosted/awesome-selfhosted-data"},
   "payload":{"review":{"state":"approved","html_url":"https://gh/r/3038"},
              "pull_request":{"number":3038}}},
  {"type":"PushEvent","created_at":"2026-09-06T11:00:00Z",
   "repo":{"name":"someone/else"},"payload":{}},
  {"type":"PullRequestReviewEvent","created_at":"2026-09-06T10:00:00Z",
   "repo":{"name":"awesome-selfhosted/awesome-selfhosted-data"},
   "payload":{"review":{"state":"commented","html_url":"https://gh/r/3038b"},
              "pull_request":{"number":3038}}},
  {"type":"PullRequestEvent","created_at":"2026-09-05T09:00:00Z",
   "repo":{"name":"Rabenherz112/Rabenherz112"},
   "payload":{"action":"closed","pull_request":{"number":1,"merged":true,"html_url":"https://gh/p/1"}}},
  {"type":"PullRequestEvent","created_at":"2026-09-04T09:00:00Z",
   "repo":{"name":"louislam/uptime-kuma"},
   "payload":{"action":"opened","pull_request":{"number":412,"html_url":"https://gh/p/412"}}}
]`

func TestFetchGitHub(t *testing.T) {
	var gotAuth, gotQuery string
	url := stub(t, map[string]http.HandlerFunc{
		"/users/Rabenherz112/events/public": func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotQuery = r.URL.RawQuery
			replyJSON(githubEvents)(w, r)
		},
	})

	got, err := FetchGitHub(ctx(t), GitHubConfig{
		BaseURL: url,
		User:    "Rabenherz112",
		Token:   "t0ken",
		Ignore:  []string{"Rabenherz112/Rabenherz112"},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("FetchGitHub: %v", err)
	}

	if gotAuth != "Bearer t0ken" {
		t.Errorf("Authorization = %q, want a bearer token", gotAuth)
	}
	if !strings.Contains(gotQuery, "per_page=100") {
		t.Errorf("query = %q, want per_page=100", gotQuery)
	}

	// Expected: the push is not renderable, the profile repo is ignored, and
	// the second event on #3038 is folded into the first so one pull request
	// cannot take two lines.
	if len(got.Items) != 2 {
		t.Fatalf("got %d items, want 2: %+v", len(got.Items), got.Items)
	}
	if got.Items[0].Verb != "approved" || got.Items[0].Num != "#3038" {
		t.Errorf("first item = %+v", got.Items[0])
	}
	if got.Items[1].Num != "#412" {
		t.Errorf("second item = %+v", got.Items[1])
	}
	// Newest first, whatever order the API used.
	if got.Items[0].When.Before(got.Items[1].When) {
		t.Error("items are not newest first")
	}
	if got.FetchedAt.IsZero() {
		t.Error("FetchedAt was not stamped")
	}
}

func TestFetchGitHubLimit(t *testing.T) {
	url := stub(t, map[string]http.HandlerFunc{
		"/users/x/events/public": replyJSON(githubEvents),
	})
	got, err := FetchGitHub(ctx(t), GitHubConfig{BaseURL: url, User: "x", Limit: 1})
	if err != nil {
		t.Fatalf("FetchGitHub: %v", err)
	}
	if len(got.Items) != 1 {
		t.Errorf("got %d items, want the limit of 1", len(got.Items))
	}
}

func TestFetchGitHubFailure(t *testing.T) {
	url := stub(t, map[string]http.HandlerFunc{
		"/users/x/events/public": fails(http.StatusForbidden),
	})
	if _, err := FetchGitHub(ctx(t), GitHubConfig{BaseURL: url, User: "x", Limit: 10}); err == nil {
		t.Fatal("want an error on 403, got nil")
	}
}

// ---------------------------------------------------------------- Wakapi ---

const wakapiStats = `{"data":{"languages":[
  {"name":"Go","percent":16.0},
  {"name":"PowerShell","percent":34.0},
  {"name":"Other","percent":30.0},
  {"name":"JavaScript","percent":27.0},
  {"name":"Zero","percent":0}
]}}`

const wakapiSummaries = `{"data":[
  {"range":{"date":"2026-09-05"},"grand_total":{"total_seconds":3600},
   "projects":[{"name":"profile","total_seconds":3600}]},
  {"range":{"date":"2026-09-06"},"grand_total":{"total_seconds":1800},
   "projects":[{"name":"profile","total_seconds":900},{"name":"other","total_seconds":900},
               {"name":"idle","total_seconds":0}]}
]}`

func TestFetchLanguages(t *testing.T) {
	var gotAuth string
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/stats/last_year": func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			replyJSON(wakapiStats)(w, r)
		},
	})

	got, err := FetchLanguages(ctx(t), WakapiConfig{BaseURL: url, APIKey: "k3y", TopLanguages: 5})
	if err != nil {
		t.Fatalf("FetchLanguages: %v", err)
	}

	// WakaTime authenticates with HTTP Basic, the API key being the whole
	// credential: base64("k3y") is "azN5".
	if gotAuth != "Basic azN5" {
		t.Errorf("Authorization = %q, want Basic azN5", gotAuth)
	}

	// Sorted by share; "Other" is a bucket rather than a language and a
	// zero-percent entry is not worth a row.
	want := []string{"PowerShell", "JavaScript", "Go"}
	if len(got.Items) != len(want) {
		t.Fatalf("got %d languages, want %d: %+v", len(got.Items), len(want), got.Items)
	}
	for i, name := range want {
		if got.Items[i].Name != name {
			t.Errorf("language %d = %q, want %q", i, got.Items[i].Name, name)
		}
	}
}

func TestFetchLanguagesEmptyIsAnError(t *testing.T) {
	// A instance that answers but has nothing to say must not be allowed to
	// wipe the card; it has to read as a failure so the old values are kept.
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/stats/last_year": replyJSON(`{"data":{"languages":[]}}`),
	})
	if _, err := FetchLanguages(ctx(t), WakapiConfig{BaseURL: url, APIKey: "k", TopLanguages: 5}); err == nil {
		t.Fatal("want an error for an empty language list, got nil")
	}
}

func TestFetchCoding(t *testing.T) {
	var gotQuery string
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/summaries": func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			replyJSON(wakapiSummaries)(w, r)
		},
	})

	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := FetchCoding(ctx(t), WakapiConfig{BaseURL: url, APIKey: "k", ChartDays: 2}, now)
	if err != nil {
		t.Fatalf("FetchCoding: %v", err)
	}

	// A two-day window ends today and starts yesterday: both ends inclusive.
	if !strings.Contains(gotQuery, "start=2026-09-05") || !strings.Contains(gotQuery, "end=2026-09-06") {
		t.Errorf("query = %q, want a 2026-09-05..2026-09-06 window", gotQuery)
	}
	if got.TotalSeconds != 5400 {
		t.Errorf("TotalSeconds = %d, want 5400", got.TotalSeconds)
	}
	// Distinct projects across the window, not per day, and a project with no
	// time recorded does not count.
	if got.Projects != 2 {
		t.Errorf("Projects = %d, want 2", got.Projects)
	}
	if len(got.Days) != 2 || got.Days[0].Date != "2026-09-05" {
		t.Errorf("Days = %+v", got.Days)
	}
}

func TestFetchCodingFailure(t *testing.T) {
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/summaries": fails(http.StatusUnauthorized),
	})
	now := time.Now()
	if _, err := FetchCoding(ctx(t), WakapiConfig{BaseURL: url, APIKey: "bad", ChartDays: 14}, now); err == nil {
		t.Fatal("want an error on 401, got nil")
	}
}

// --------------------------------------------------------------- AniList ---

const anilistOK = `{"data":{
  "User":{"statistics":{"manga":{"count":1284,"chaptersRead":96301,"volumesRead":4912,"meanScore":78}}},
  "MediaListCollection":{"lists":[{"entries":[
    {"progress":41,"updatedAt":100,"media":{"title":{"romaji":"Blame!","english":""},
      "coverImage":{"medium":"COVER/blame.jpg"}}},
    {"progress":187,"updatedAt":300,"media":{"title":{"romaji":"Vinland Saga","english":"Vinland Saga"},
      "coverImage":{"medium":"COVER/vinland.jpg"}}},
    {"progress":364,"updatedAt":200,"media":{"title":{"romaji":"Berserk","english":""},
      "coverImage":{"medium":""}}}
  ]}]}}}`

func TestFetchAniList(t *testing.T) {
	var covers int
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		replyJSON(strings.ReplaceAll(anilistOK, "COVER", srv.URL+"/img"))(w, r)
	})
	mux.HandleFunc("/img/", func(w http.ResponseWriter, _ *http.Request) {
		covers++
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0}) // a JPEG's opening bytes
	})

	got, err := FetchAniList(ctx(t), AniListConfig{BaseURL: srv.URL + "/graphql", User: "x", Reading: 3})
	if err != nil {
		t.Fatalf("FetchAniList: %v", err)
	}

	if got.SeriesTracked != 1284 || got.ChaptersRead != 96301 || got.VolumesRead != 4912 {
		t.Errorf("totals = %+v", got)
	}
	// AniList reports the mean out of 100 whatever the display preference; the
	// card shows it out of 10.
	if got.MeanScore != 7.8 {
		t.Errorf("MeanScore = %v, want 7.8", got.MeanScore)
	}

	// Most recently read first.
	if len(got.Reading) != 3 || got.Reading[0].Title != "Vinland Saga" {
		t.Fatalf("Reading = %+v", got.Reading)
	}
	if got.Reading[0].Meta != "ch. 187" {
		t.Errorf("Meta = %q, want %q", got.Reading[0].Meta, "ch. 187")
	}
	// The romaji title is used when there is no English one.
	if got.Reading[1].Title != "Berserk" {
		t.Errorf("second title = %q, want the romaji fallback", got.Reading[1].Title)
	}

	// Covers travel with the card, since it cannot fetch them when viewed.
	if len(got.Reading[0].Cover) == 0 || got.Reading[0].CoverMIME != "image/jpeg" {
		t.Error("cover art was not embedded")
	}
	// Berserk has no cover URL, so only two were fetched and it falls back to
	// a placeholder box rather than failing the source.
	if covers != 2 {
		t.Errorf("fetched %d covers, want 2", covers)
	}
	if len(got.Reading[1].Cover) != 0 {
		t.Error("a series with no cover URL should have no cover")
	}
}

func TestFetchAniListGraphQLError(t *testing.T) {
	// GraphQL reports failures in the body with a 200 status, so the errors
	// array has to be checked explicitly or a wiped card looks like success.
	url := stub(t, map[string]http.HandlerFunc{
		"/": replyJSON(`{"errors":[{"message":"User not found"}],"data":null}`),
	})
	_, err := FetchAniList(ctx(t), AniListConfig{BaseURL: url, User: "nope", Reading: 3})
	if err == nil {
		t.Fatal("want an error from the GraphQL errors array, got nil")
	}
	if !strings.Contains(err.Error(), "User not found") {
		t.Errorf("error = %v, want it to carry the API's message", err)
	}
}

func TestFetchAniListCoverFailureIsNotFatal(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		replyJSON(strings.ReplaceAll(anilistOK, "COVER", srv.URL+"/img"))(w, r)
	})
	mux.HandleFunc("/img/", fails(http.StatusNotFound))

	got, err := FetchAniList(ctx(t), AniListConfig{BaseURL: srv.URL + "/graphql", User: "x", Reading: 3})
	if err != nil {
		t.Fatalf("a missing cover should not fail the source: %v", err)
	}
	if len(got.Reading) != 3 {
		t.Errorf("got %d series, want 3 with placeholder covers", len(got.Reading))
	}
}

// ----------------------------------------------------------------- Steam ---

func TestFetchSteamResolvesVanityURL(t *testing.T) {
	var gotSteamID string
	url := stub(t, map[string]http.HandlerFunc{
		"/ISteamUser/ResolveVanityURL/v1/": replyJSON(`{"response":{"steamid":"7656119","success":1}}`),
		"/IPlayerService/GetRecentlyPlayedGames/v1/": func(w http.ResponseWriter, r *http.Request) {
			gotSteamID = r.URL.Query().Get("steamid")
			replyJSON(`{"response":{"total_count":3,"games":[
				{"appid":2,"name":"Balatro","img_icon_url":"bbb","playtime_2weeks":366},
				{"appid":1,"name":"Hades II","img_icon_url":"aaa","playtime_2weeks":744},
				{"appid":3,"name":"Noita","img_icon_url":"","playtime_2weeks":228}]}}`)(w, r)
		},
		"/steamcommunity/public/images/apps/": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
		},
		"/IPlayerService/GetOwnedGames/v1/": replyJSON(`{"response":{"game_count":341}}`),
	})

	got, err := FetchSteam(ctx(t), SteamConfig{
		BaseURL: url, MediaURL: url, APIKey: "k", VanityURL: "rabenherz", Recent: 3,
	})
	if err != nil {
		t.Fatalf("FetchSteam: %v", err)
	}
	if gotSteamID != "7656119" {
		t.Errorf("steamid = %q, want the resolved id", gotSteamID)
	}
	// The card cannot load an icon when it is viewed, so the artwork travels
	// with it. A game with no icon hash falls back to a placeholder square
	// rather than failing the source.
	if len(got.Recent[0].Icon) == 0 || got.Recent[0].IconMIME != "image/jpeg" {
		t.Error("the top game's icon was not embedded")
	}
	if len(got.Recent[2].Icon) != 0 {
		t.Error("a game with no icon hash should have no icon")
	}
	if got.OwnedGames != 341 {
		t.Errorf("OwnedGames = %d, want 341", got.OwnedGames)
	}
	// Most played over the fortnight first, so the card reads as a ranking.
	if len(got.Recent) != 3 || got.Recent[0].Name != "Hades II" {
		t.Errorf("Recent = %+v", got.Recent)
	}
}

func TestFetchSteamPrivateProfile(t *testing.T) {
	// A private profile answers 200 with an empty list rather than an error.
	// Treating that as success would blank the card, so it has to be an error.
	url := stub(t, map[string]http.HandlerFunc{
		"/IPlayerService/GetRecentlyPlayedGames/v1/": replyJSON(`{"response":{}}`),
	})
	_, err := FetchSteam(ctx(t), SteamConfig{BaseURL: url, APIKey: "k", SteamID: "1", Recent: 3})
	if err == nil {
		t.Fatal("want an error for an empty game list, got nil")
	}
	if !strings.Contains(err.Error(), "public") {
		t.Errorf("error = %v, want it to hint at profile privacy", err)
	}
}

func TestFetchSteamOwnedGamesFailureIsNotFatal(t *testing.T) {
	// The library size is a caption on the card. Losing it should not cost the
	// rest of the card.
	url := stub(t, map[string]http.HandlerFunc{
		"/IPlayerService/GetRecentlyPlayedGames/v1/": replyJSON(
			`{"response":{"games":[{"name":"Noita","playtime_2weeks":228}]}}`),
		"/IPlayerService/GetOwnedGames/v1/": fails(http.StatusInternalServerError),
	})
	got, err := FetchSteam(ctx(t), SteamConfig{BaseURL: url, APIKey: "k", SteamID: "1", Recent: 3})
	if err != nil {
		t.Fatalf("FetchSteam: %v", err)
	}
	if len(got.Recent) != 1 {
		t.Errorf("Recent = %+v, want the games despite the owned-count failure", got.Recent)
	}
	if got.OwnedGames != 0 {
		t.Errorf("OwnedGames = %d, want 0 when the count could not be had", got.OwnedGames)
	}
}

func TestFetchSteamUnresolvableVanity(t *testing.T) {
	url := stub(t, map[string]http.HandlerFunc{
		"/ISteamUser/ResolveVanityURL/v1/": replyJSON(`{"response":{"success":42}}`),
	})
	if _, err := FetchSteam(ctx(t), SteamConfig{BaseURL: url, APIKey: "k", VanityURL: "ghost", Recent: 3}); err == nil {
		t.Fatal("want an error when the vanity name does not resolve, got nil")
	}
}

// ---------------------------------------------------------------- Kavita ---

func TestFetchKavita(t *testing.T) {
	var gotBearer string
	url := stub(t, map[string]http.HandlerFunc{
		"/api/Plugin/authenticate": replyJSON(`{"token":"jwt-123","username":"rabenherz"}`),
		"/api/Stats/server/stats": func(w http.ResponseWriter, r *http.Request) {
			gotBearer = r.Header.Get("Authorization")
			replyJSON(`{"seriesCount":1284,"totalSize":2638827906662,"totalGenres":41,"chapterCount":96301}`)(w, r)
		},
	})

	got, err := FetchKavita(ctx(t), KavitaConfig{BaseURL: url, APIKey: "k"})
	if err != nil {
		t.Fatalf("FetchKavita: %v", err)
	}
	// Kavita has no long-lived token: the API key buys a short-lived JWT which
	// then authorises the statistics call.
	if gotBearer != "Bearer jwt-123" {
		t.Errorf("Authorization = %q, want the JWT from the plugin endpoint", gotBearer)
	}
	if got.Series != 1284 || got.Bytes != 2638827906662 || got.Genres != 41 {
		t.Errorf("stats = %+v", got)
	}
}

// TestFetchKavitaUnreachable is the case that matters most for a self-hosted
// service: the box is off, or the generator cannot route to it. The source has
// to report an error rather than returning zeroes, because zeroes would be
// written to the snapshot and the card would go blank.
func TestFetchKavitaUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NewServeMux())
	down := srv.URL
	srv.Close() // nothing is listening on that port any more

	got, err := FetchKavita(ctx(t), KavitaConfig{BaseURL: down, APIKey: "k"})
	if err == nil {
		t.Fatal("want an error when the instance is unreachable, got nil")
	}
	if got.Series != 0 || !got.FetchedAt.IsZero() {
		t.Errorf("a failed fetch must not return usable values: %+v", got)
	}
}

func TestFetchKavitaBadAuth(t *testing.T) {
	url := stub(t, map[string]http.HandlerFunc{
		"/api/Plugin/authenticate": fails(http.StatusUnauthorized),
	})
	if _, err := FetchKavita(ctx(t), KavitaConfig{BaseURL: url, APIKey: "wrong"}); err == nil {
		t.Fatal("want an error on a rejected API key, got nil")
	}
}

func TestFetchKavitaEmptyStats(t *testing.T) {
	// A Kavita that authenticates but reports nothing is more likely a schema
	// change than an empty library, and either way must not blank the card.
	url := stub(t, map[string]http.HandlerFunc{
		"/api/Plugin/authenticate": replyJSON(`{"token":"j"}`),
		"/api/Stats/server/stats":  replyJSON(`{}`),
	})
	if _, err := FetchKavita(ctx(t), KavitaConfig{BaseURL: url, APIKey: "k"}); err == nil {
		t.Fatal("want an error for an empty statistics response, got nil")
	}
}

// TestSlowSourceIsCutOff checks the context deadline is honoured, so one
// unresponsive self-hosted service cannot hold the whole workflow open.
func TestSlowSourceIsCutOff(t *testing.T) {
	// The handler stalls until the test releases it. It cannot simply wait on
	// the request context: httptest.Server.Close blocks until every handler has
	// returned, and a server that never notices the client hung up would wedge
	// the test rather than fail it.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	c, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := FetchKavita(c, KavitaConfig{BaseURL: srv.URL, APIKey: "k"}); err == nil {
		t.Fatal("want an error when the deadline passes, got nil")
	}
	// The point is that it gave up on the deadline rather than on the client's
	// own much longer timeout.
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %v to give up, want the context deadline to cut it off", elapsed)
	}
}

// TestFetchCodingWithWakapiRepeatedDate covers the shape a real Wakapi instance
// returns: range.date is the moment the request was made, identical on every
// entry, rather than the day being summarised. Reading it straight put a
// timestamp under every bar of the chart.
func TestFetchCodingWithWakapiRepeatedDate(t *testing.T) {
	const sameStamp = "2026-09-06T17:43:39+02:00"
	body := `{"data":[
		{"range":{"date":"` + sameStamp + `","start":"` + sameStamp + `"},"grand_total":{"total_seconds":100},"projects":[]},
		{"range":{"date":"` + sameStamp + `","start":"` + sameStamp + `"},"grand_total":{"total_seconds":200},"projects":[]},
		{"range":{"date":"` + sameStamp + `","start":"` + sameStamp + `"},"grand_total":{"total_seconds":300},"projects":[]}
	]}`
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/summaries": replyJSON(body),
	})

	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := FetchCoding(ctx(t), WakapiConfig{BaseURL: url, APIKey: "k", ChartDays: 3}, now)
	if err != nil {
		t.Fatalf("FetchCoding: %v", err)
	}

	// Three days ending today, counted from the start of the window that was
	// requested rather than from the useless field.
	want := []string{"2026-09-04", "2026-09-05", "2026-09-06"}
	if len(got.Days) != len(want) {
		t.Fatalf("got %d days, want %d", len(got.Days), len(want))
	}
	for i, w := range want {
		if got.Days[i].Date != w {
			t.Errorf("day %d = %q, want %q", i, got.Days[i].Date, w)
		}
	}
	// The totals are fine in the real response; only the dates were not.
	if got.Days[2].Seconds != 300 {
		t.Errorf("seconds were disturbed: %+v", got.Days)
	}
}

// TestFetchCodingPrefersRealDates checks that a server filling the field
// correctly, as WakaTime itself does, is believed rather than overridden.
func TestFetchCodingPrefersRealDates(t *testing.T) {
	body := `{"data":[
		{"range":{"date":"2026-01-10"},"grand_total":{"total_seconds":100},"projects":[]},
		{"range":{"date":"2026-01-11"},"grand_total":{"total_seconds":200},"projects":[]}
	]}`
	url := stub(t, map[string]http.HandlerFunc{
		"/api/compat/wakatime/v1/users/current/summaries": replyJSON(body),
	})

	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := FetchCoding(ctx(t), WakapiConfig{BaseURL: url, APIKey: "k", ChartDays: 2}, now)
	if err != nil {
		t.Fatalf("FetchCoding: %v", err)
	}
	if got.Days[0].Date != "2026-01-10" || got.Days[1].Date != "2026-01-11" {
		t.Errorf("dates = %+v, want the ones the API reported", got.Days)
	}
}
