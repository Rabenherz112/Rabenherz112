package sources

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
)

// SteamAPI is the Steam Web API root, and SteamMedia is where the game icons
// live. They are different hosts, so they are configured separately.
const (
	SteamAPI   = "https://api.steampowered.com"
	SteamMedia = "https://media.steampowered.com"
)

// SteamConfig holds the Steam Web API credentials.
type SteamConfig struct {
	APIKey string
	// BaseURL overrides the API root and MediaURL the icon host. Empty means
	// SteamAPI and SteamMedia; tests point them at a local server.
	BaseURL  string
	MediaURL string
	// SteamID is the 64-bit id. When it is empty, VanityURL is resolved to one.
	SteamID string
	// VanityURL is the name in a steamcommunity.com/id/<name> profile link.
	VanityURL string
	// Recent is how many recently played games the card has room for.
	Recent int
}

type steamVanity struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
	} `json:"response"`
}

type steamRecent struct {
	Response struct {
		Games []struct {
			AppID           int    `json:"appid"`
			Name            string `json:"name"`
			ImgIconURL      string `json:"img_icon_url"`
			Playtime2Weeks  int    `json:"playtime_2weeks"`
			PlaytimeForever int    `json:"playtime_forever"`
		} `json:"games"`
	} `json:"response"`
}

// steamIconURL builds the URL of a game's 32px store icon. The API returns only
// the filename half of it; the rest is a fixed path on Steam's media host.
func steamIconURL(base string, appID int, hash string) string {
	if appID == 0 || hash == "" {
		return ""
	}
	return fmt.Sprintf("%s/steamcommunity/public/images/apps/%d/%s.jpg", base, appID, hash)
}

// maxIconBytes caps a single game icon. They are 32x32 JPEGs of a couple of
// kilobytes; the limit is here so an unexpected redirect cannot bloat the card.
const maxIconBytes = 128 << 10

type steamOwned struct {
	Response struct {
		GameCount int `json:"game_count"`
	} `json:"response"`
}

// FetchSteam returns the games played in the last two weeks and the size of the
// library.
//
// Both endpoints need the profile's game details to be public. If they are set
// to private, Steam answers 200 with an empty response rather than an error,
// which is reported here as a failure so the card keeps its previous values
// instead of rendering empty.
func FetchSteam(ctx context.Context, cfg SteamConfig) (model.Steam, error) {
	var out model.Steam

	id := cfg.SteamID
	if id == "" {
		if cfg.VanityURL == "" {
			return out, fmt.Errorf("steam: neither a steam id nor a vanity url was configured")
		}
		var v steamVanity
		u := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s",
			base(cfg.BaseURL, SteamAPI), url.QueryEscape(cfg.APIKey), url.QueryEscape(cfg.VanityURL))
		if err := getJSON(ctx, u, nil, &v); err != nil {
			return out, err
		}
		if v.Response.Success != 1 || v.Response.SteamID == "" {
			return out, fmt.Errorf("steam: could not resolve vanity url %q", cfg.VanityURL)
		}
		id = v.Response.SteamID
	}

	var recent steamRecent
	u := fmt.Sprintf("%s/IPlayerService/GetRecentlyPlayedGames/v1/?key=%s&steamid=%s",
		base(cfg.BaseURL, SteamAPI), url.QueryEscape(cfg.APIKey), url.QueryEscape(id))
	if err := getJSON(ctx, u, nil, &recent); err != nil {
		return out, err
	}
	if len(recent.Response.Games) == 0 {
		return out, fmt.Errorf("steam: no recently played games returned (is the profile public?)")
	}

	games := recent.Response.Games
	sort.SliceStable(games, func(i, j int) bool {
		return games[i].Playtime2Weeks > games[j].Playtime2Weeks
	})
	for _, g := range games {
		if len(out.Recent) == cfg.Recent {
			break
		}
		game := model.Game{
			Name:           g.Name,
			TwoWeekMinutes: g.Playtime2Weeks,
		}
		if u := steamIconURL(base(cfg.MediaURL, SteamMedia), g.AppID, g.ImgIconURL); u != "" {
			// A missing icon only costs a placeholder square, so a failure
			// here is not worth failing the whole source over.
			if b, mime, err := getBytes(ctx, u, maxIconBytes); err == nil {
				game.Icon, game.IconMIME = b, mime
			}
		}
		out.Recent = append(out.Recent, game)
	}

	// The library size is a nice-to-have on the card, so a failure here only
	// loses the count rather than the whole source.
	var owned steamOwned
	u = fmt.Sprintf("%s/IPlayerService/GetOwnedGames/v1/?key=%s&steamid=%s",
		base(cfg.BaseURL, SteamAPI), url.QueryEscape(cfg.APIKey), url.QueryEscape(id))
	if err := getJSON(ctx, u, nil, &owned); err == nil {
		out.OwnedGames = owned.Response.GameCount
	}

	out.FetchedAt = fetchedNow()
	return out, nil
}
