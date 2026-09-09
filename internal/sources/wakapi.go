package sources

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
)

// WakapiConfig points at a Wakapi instance. Wakapi serves the WakaTime API
// under /api/compat/wakatime/v1, so the same code would work against
// wakatime.com by changing BaseURL.
type WakapiConfig struct {
	BaseURL string
	APIKey  string
	// TopLanguages is how many rows the languages card has room for.
	TopLanguages int
	// ChartDays is the width of the coding-time chart, in days.
	ChartDays int
}

// wakaAuth builds the WakaTime authorization header: HTTP Basic with the API
// key as the whole credential.
func wakaAuth(key string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key)))
	return h
}

type wakaStats struct {
	Data struct {
		Languages []struct {
			Name    string  `json:"name"`
			Percent float64 `json:"percent"`
		} `json:"languages"`
	} `json:"data"`
}

type wakaSummaries struct {
	Data []struct {
		Range struct {
			Date  string `json:"date"`
			Start string `json:"start"`
		} `json:"range"`
		GrandTotal struct {
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"grand_total"`
		Projects []struct {
			Name         string  `json:"name"`
			TotalSeconds float64 `json:"total_seconds"`
		} `json:"projects"`
	} `json:"data"`
}

// FetchLanguages returns the share of coding time per language over the last
// year, largest first.
func FetchLanguages(ctx context.Context, cfg WakapiConfig) (model.Languages, error) {
	var out model.Languages

	url := base(cfg.BaseURL, "") + "/api/compat/wakatime/v1/users/current/stats/last_year"
	var stats wakaStats
	if err := getJSON(ctx, url, wakaAuth(cfg.APIKey), &stats); err != nil {
		return out, err
	}

	langs := stats.Data.Languages
	sort.SliceStable(langs, func(i, j int) bool { return langs[i].Percent > langs[j].Percent })

	for _, l := range langs {
		if len(out.Items) == cfg.TopLanguages {
			break
		}
		// Wakapi reports time spent in files it could not classify under this
		// name. It is not a language and it crowds out real rows.
		if strings.EqualFold(l.Name, "Other") || strings.EqualFold(l.Name, "Unknown") {
			continue
		}
		if l.Percent <= 0 {
			continue
		}
		out.Items = append(out.Items, model.Language{Name: l.Name, Pct: l.Percent})
	}
	if len(out.Items) == 0 {
		return out, fmt.Errorf("wakapi returned no languages")
	}

	out.FetchedAt = fetchedNow()
	return out, nil
}

// FetchCoding returns the daily coding totals over the chart window, along with
// the number of distinct projects touched in it.
func FetchCoding(ctx context.Context, cfg WakapiConfig, now time.Time) (model.Coding, error) {
	var out model.Coding

	// The window is inclusive at both ends, so it spans ChartDays days counting
	// today as the last one.
	end := now.UTC()
	start := end.AddDate(0, 0, -(cfg.ChartDays - 1))

	url := fmt.Sprintf("%s/api/compat/wakatime/v1/users/current/summaries?start=%s&end=%s",
		base(cfg.BaseURL, ""),
		start.Format("2006-01-02"), end.Format("2006-01-02"))

	var sum wakaSummaries
	if err := getJSON(ctx, url, wakaAuth(cfg.APIKey), &sum); err != nil {
		return out, err
	}
	if len(sum.Data) == 0 {
		return out, fmt.Errorf("wakapi returned no summaries for %s..%s",
			start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	dates := summaryDates(sum, start)

	projects := map[string]bool{}
	for i, d := range sum.Data {
		out.Days = append(out.Days, model.Day{
			Date:    dates[i],
			Seconds: int(d.GrandTotal.TotalSeconds),
		})
		out.TotalSeconds += int(d.GrandTotal.TotalSeconds)
		for _, p := range d.Projects {
			if p.TotalSeconds > 0 {
				projects[p.Name] = true
			}
		}
	}
	out.Projects = len(projects)
	out.FetchedAt = fetchedNow()
	return out, nil
}

// summaryDates works out which day each summary in the response covers.
//
// It cannot simply trust the field that nominally holds it. Wakapi fills
// range.date with the moment the request was made rather than the day being
// summarised, so a fortnight of summaries comes back carrying fourteen copies
// of the same timestamp. Reading that as the date put the request time under
// the chart instead of the day.
//
// The summaries are returned in order across the window that was asked for, so
// counting days from the start of that window is the reliable answer. The
// field is still preferred when it parses to distinct dates, because WakaTime
// itself fills it correctly and its own dates are authoritative.
func summaryDates(sum wakaSummaries, start time.Time) []string {
	fromAPI := make([]string, len(sum.Data))
	seen := make(map[string]bool, len(sum.Data))
	usable := len(sum.Data) > 0

	for i, d := range sum.Data {
		day := parseWakaDay(d.Range.Start)
		if day == "" {
			day = parseWakaDay(d.Range.Date)
		}
		if day == "" || seen[day] {
			usable = false
			break
		}
		seen[day] = true
		fromAPI[i] = day
	}
	if usable {
		return fromAPI
	}

	out := make([]string, len(sum.Data))
	for i := range sum.Data {
		out[i] = start.AddDate(0, 0, i).Format(dayLayout)
	}
	return out
}

// dayLayout is how a day is stored in the snapshot, whatever shape the API
// used to express it.
const dayLayout = "2006-01-02"

// parseWakaDay normalises a date from the API, which may be a plain day or a
// full timestamp, to a day. It returns an empty string for anything else.
func parseWakaDay(s string) string {
	if s == "" {
		return ""
	}
	for _, layout := range []string{dayLayout, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format(dayLayout)
		}
	}
	return ""
}
