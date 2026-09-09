package sources

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Rabenherz112/Rabenherz112/internal/model"
)

// GitHubAPI is the public API root, and the default for GitHubConfig.BaseURL.
const GitHubAPI = "https://api.github.com"

// GitHubConfig selects whose activity to show.
type GitHubConfig struct {
	User  string
	Token string
	// BaseURL overrides the API root. Empty means GitHubAPI; tests point it at
	// a local server.
	BaseURL string
	// Ignore lists repositories to leave out, in owner/name form. The profile
	// repository is in here by default: its own scheduled commits would
	// otherwise crowd out everything worth showing.
	Ignore []string
	// Limit is how many events the card has room for.
	Limit int
}

// ghEvent is the part of a public event this card uses. The GitHub events API
// packs a different payload shape behind each event type, so the union is
// decoded and the relevant fields picked out per type.
type ghEvent struct {
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Repo      struct {
		Name string `json:"name"` // owner/name
	} `json:"repo"`
	Payload struct {
		Action string `json:"action"`
		Review *struct {
			State   string `json:"state"`
			HTMLURL string `json:"html_url"`
		} `json:"review"`
		PullRequest *struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
			Merged  bool   `json:"merged"`
		} `json:"pull_request"`
		Issue *struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
		} `json:"issue"`
		Release *struct {
			TagName string `json:"tag_name"`
			Name    string `json:"name"`
			HTMLURL string `json:"html_url"`
		} `json:"release"`
		Member *struct {
			Login string `json:"login"`
		} `json:"member"`
	} `json:"payload"`
}

// FetchGitHub returns the most recent public reviews and pull requests.
//
// This reads the public events feed, which only covers roughly the last 90 days
// and 300 events. That is well past what the card shows, and it is the only
// endpoint that reports reviews and pull requests together in one pass.
func FetchGitHub(ctx context.Context, cfg GitHubConfig) (model.Activity, error) {
	var out model.Activity

	header := http.Header{}
	header.Set("Accept", "application/vnd.github+json")
	header.Set("X-GitHub-Api-Version", "2022-11-28")
	if cfg.Token != "" {
		header.Set("Authorization", "Bearer "+cfg.Token)
	}

	url := fmt.Sprintf("%s/users/%s/events/public?per_page=100", base(cfg.BaseURL, GitHubAPI), cfg.User)
	var events []ghEvent
	if err := getJSON(ctx, url, header, &events); err != nil {
		return out, err
	}

	ignored := make(map[string]bool, len(cfg.Ignore))
	for _, r := range cfg.Ignore {
		ignored[strings.ToLower(r)] = true
	}

	// Events arrive newest first. Keep one line per pull request so a burst of
	// activity on a single PR cannot fill the card.
	seen := make(map[string]bool)
	for _, e := range events {
		if ignored[strings.ToLower(e.Repo.Name)] {
			continue
		}
		ev, ok := convertEvent(e, cfg.User)
		if !ok {
			continue
		}
		key := e.Repo.Name + ev.Num
		if seen[key] {
			continue
		}
		seen[key] = true

		out.Items = append(out.Items, ev)
		if len(out.Items) == cfg.Limit {
			break
		}
	}

	sort.SliceStable(out.Items, func(i, j int) bool {
		return out.Items[i].When.After(out.Items[j].When)
	})
	out.FetchedAt = fetchedNow()
	return out, nil
}

// convertEvent maps one raw event to a card line, reporting false for the event
// types the card does not show.
//
// The card shows work that reached a conclusion: pull requests and issues
// opened, closed or merged, reviews that approved or asked for changes,
// releases, and becoming a collaborator somewhere. Comments are deliberately
// absent. They are the easiest events to generate and the least informative to
// read, and they would crowd out the reviews on a profile whose owner reviews a
// great deal.
//
// user is whose feed this is, which a MemberEvent needs in order to mean what
// the card says it means. See that case.
func convertEvent(e ghEvent, user string) (model.Event, bool) {
	ev := model.Event{Repo: e.Repo.Name, When: e.CreatedAt}

	switch e.Type {
	case "PullRequestReviewEvent":
		if e.Payload.Review == nil || e.Payload.PullRequest == nil {
			return ev, false
		}
		switch e.Payload.Review.State {
		case "approved":
			ev.Verb = "approved"
		case "changes_requested":
			ev.Verb = "changes"
		default:
			// A review left as a plain comment, or dismissed, says nothing
			// about the outcome.
			return ev, false
		}
		ev.Num = fmt.Sprintf("#%d", e.Payload.PullRequest.Number)
		ev.URL = e.Payload.Review.HTMLURL

	case "PullRequestEvent":
		if e.Payload.PullRequest == nil {
			return ev, false
		}
		switch e.Payload.Action {
		case "opened", "reopened":
			ev.Verb = "opened"
		case "closed":
			ev.Verb = "closed"
			if e.Payload.PullRequest.Merged {
				ev.Verb = "merged"
			}
		default:
			return ev, false
		}
		ev.Num = fmt.Sprintf("#%d", e.Payload.PullRequest.Number)
		ev.URL = e.Payload.PullRequest.HTMLURL

	case "IssuesEvent":
		if e.Payload.Issue == nil {
			return ev, false
		}
		switch e.Payload.Action {
		case "opened", "reopened":
			ev.Verb = "opened"
		case "closed":
			ev.Verb = "closed"
		default:
			return ev, false
		}
		ev.Num = fmt.Sprintf("#%d", e.Payload.Issue.Number)
		ev.URL = e.Payload.Issue.HTMLURL

	case "MemberEvent":
		// This event fires when a collaborator is added to a repository, and
		// GitHub makes the actor the person who did the adding, not the person
		// added. In a maintainer's own feed it therefore covers both "I was
		// added here" and "I added someone here", which are not the same
		// statement at all. Only the first belongs on the card, so the member
		// has to be this profile's owner.
		if e.Payload.Member == nil || e.Payload.Action != "added" {
			return ev, false
		}
		if !strings.EqualFold(e.Payload.Member.Login, user) {
			return ev, false
		}
		ev.Verb = "joined"
		// There is no number to show; the repository is the whole story.
		ev.URL = "https://github.com/" + e.Repo.Name

	case "ReleaseEvent":
		// Only a published release counts. Drafts and edits are bookkeeping.
		if e.Payload.Release == nil || e.Payload.Action != "published" {
			return ev, false
		}
		ev.Verb = "released"
		ev.Num = e.Payload.Release.TagName
		ev.URL = e.Payload.Release.HTMLURL

	default:
		return ev, false
	}

	return ev, true
}
