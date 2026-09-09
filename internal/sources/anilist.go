package sources

import (
	"context"
	"fmt"
	"sort"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
)

// AniListConfig selects the AniList profile to read. The endpoint is public and
// needs no key for a public profile.
type AniListConfig struct {
	User string
	// Reading is how many currently-read series the card has room for.
	Reading int
	// BaseURL overrides the GraphQL endpoint. Empty means AniListAPI; tests
	// point it at a local server.
	BaseURL string
}

// AniListAPI is the public GraphQL endpoint, and the default for
// AniListConfig.BaseURL.
const AniListAPI = "https://graphql.anilist.co"

// anilistQuery pulls the lifetime manga totals and the in-progress list in one
// round trip. Only the medium cover is requested: it is displayed about 52px
// wide, and a larger one would bloat every card it is inlined into.
const anilistQuery = `query ($name: String) {
  User(name: $name) {
    statistics {
      manga { count chaptersRead volumesRead meanScore }
    }
  }
  MediaListCollection(userName: $name, type: MANGA, status: CURRENT) {
    lists {
      entries {
        progress
        updatedAt
        media { title { romaji english } coverImage { medium } }
      }
    }
  }
}`

type anilistResponse struct {
	Data struct {
		User struct {
			Statistics struct {
				Manga struct {
					Count        int     `json:"count"`
					ChaptersRead int     `json:"chaptersRead"`
					VolumesRead  int     `json:"volumesRead"`
					MeanScore    float64 `json:"meanScore"`
				} `json:"manga"`
			} `json:"statistics"`
		} `json:"User"`
		MediaListCollection struct {
			Lists []struct {
				Entries []struct {
					Progress  int   `json:"progress"`
					UpdatedAt int64 `json:"updatedAt"`
					Media     struct {
						Title struct {
							Romaji  string `json:"romaji"`
							English string `json:"english"`
						} `json:"title"`
						CoverImage struct {
							Medium string `json:"medium"`
						} `json:"coverImage"`
					} `json:"media"`
				} `json:"entries"`
			} `json:"lists"`
		} `json:"MediaListCollection"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// maxCoverBytes caps a single cover. Medium covers run well under this; the
// limit is here so an unexpected redirect cannot bloat the card.
const maxCoverBytes = 256 << 10

// FetchAniList returns the manga totals and the series currently being read,
// with their cover art already downloaded for inlining.
func FetchAniList(ctx context.Context, cfg AniListConfig) (model.AniList, error) {
	var out model.AniList

	body := map[string]any{
		"query":     anilistQuery,
		"variables": map[string]any{"name": cfg.User},
	}
	var resp anilistResponse
	if err := postJSON(ctx, base(cfg.BaseURL, AniListAPI), nil, body, &resp); err != nil {
		return out, err
	}
	// GraphQL reports failures in the body with a 200 status, so the errors
	// array has to be checked explicitly.
	if len(resp.Errors) > 0 {
		return out, fmt.Errorf("anilist: %s", resp.Errors[0].Message)
	}

	st := resp.Data.User.Statistics.Manga
	out.SeriesTracked = st.Count
	out.ChaptersRead = st.ChaptersRead
	out.VolumesRead = st.VolumesRead
	// AniList reports the mean on a 100-point scale whatever the user's display
	// preference; the card shows it out of 10.
	out.MeanScore = st.MeanScore / 10

	type entry struct {
		title     string
		progress  int
		cover     string
		updatedAt int64
	}
	var entries []entry
	for _, l := range resp.Data.MediaListCollection.Lists {
		for _, e := range l.Entries {
			title := e.Media.Title.English
			if title == "" {
				title = e.Media.Title.Romaji
			}
			if title == "" {
				continue
			}
			entries = append(entries, entry{
				title:     title,
				progress:  e.Progress,
				cover:     e.Media.CoverImage.Medium,
				updatedAt: e.UpdatedAt,
			})
		}
	}
	// Most recently read first, so the card shows what is actually in progress
	// rather than whatever order the list happens to be stored in.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].updatedAt > entries[j].updatedAt
	})

	for _, e := range entries {
		if len(out.Reading) == cfg.Reading {
			break
		}
		m := model.Media{
			Title: e.title,
			Meta:  fmt.Sprintf("ch. %d", e.progress),
		}
		if e.cover != "" {
			// A missing cover only costs a placeholder box, so a failure here
			// is not worth failing the whole source over.
			if b, mime, err := getBytes(ctx, e.cover, maxCoverBytes); err == nil {
				m.Cover, m.CoverMIME = b, mime
			}
		}
		out.Reading = append(out.Reading, m)
	}

	out.FetchedAt = fetchedNow()
	return out, nil
}
