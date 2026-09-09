package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
)

// KavitaConfig points at a Kavita instance. The API key is the one from the
// account's user settings, not a password.
type KavitaConfig struct {
	BaseURL string
	APIKey  string
	// Host is what the card prints as the library's address. It defaults to the
	// host part of BaseURL, but can differ when the generator reaches Kavita on
	// an internal address.
	Host string
}

type kavitaAuth struct {
	Token string `json:"token"`
}

// kavitaStats mirrors Kavita's ServerStatisticsDto. Only the three figures the
// card shows are decoded; the endpoint returns considerably more.
type kavitaStats struct {
	SeriesCount  int   `json:"seriesCount"`
	TotalSize    int64 `json:"totalSize"`
	TotalGenres  int   `json:"totalGenres"`
	ChapterCount int   `json:"chapterCount"`
}

// FetchKavita returns the size of the self-hosted library.
//
// Kavita has no long-lived bearer token: the API key is exchanged for a
// short-lived JWT on each run through the plugin endpoint, and that JWT
// authorises the statistics call. The statistics endpoint is admin-only, so the
// key has to belong to an account with admin rights.
func FetchKavita(ctx context.Context, cfg KavitaConfig) (model.Kavita, error) {
	var out model.Kavita

	root := base(cfg.BaseURL, "")

	var auth kavitaAuth
	authURL := fmt.Sprintf("%s/api/Plugin/authenticate?apiKey=%s&pluginName=%s",
		root, url.QueryEscape(cfg.APIKey), url.QueryEscape("profile-cards"))
	if err := postJSON(ctx, authURL, nil, nil, &auth); err != nil {
		return out, fmt.Errorf("kavita authenticate: %w", err)
	}
	if auth.Token == "" {
		return out, fmt.Errorf("kavita authenticate: no token returned")
	}

	h := http.Header{}
	h.Set("Authorization", "Bearer "+auth.Token)

	var stats kavitaStats
	if err := getJSON(ctx, root+"/api/Stats/server/stats", h, &stats); err != nil {
		return out, fmt.Errorf("kavita stats: %w", err)
	}
	if stats.SeriesCount == 0 && stats.TotalSize == 0 {
		return out, fmt.Errorf("kavita stats: empty response")
	}

	out.Series = stats.SeriesCount
	out.Bytes = stats.TotalSize
	out.Genres = stats.TotalGenres

	out.FetchedAt = fetchedNow()
	return out, nil
}
