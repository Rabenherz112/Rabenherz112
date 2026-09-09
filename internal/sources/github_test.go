package sources

import (
	"encoding/json"
	"testing"
)

// feedOwner is whose public feed the test events belong to.
const feedOwner = "Rabenherz112"

// decodeEvent builds a raw event from JSON, so these cases exercise the same
// decoding path the API response goes through rather than a hand-built struct.
func decodeEvent(t *testing.T, raw string) ghEvent {
	t.Helper()
	var e ghEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return e
}

func TestConvertEvent(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		ok   bool
		verb string
		num  string
		url  string
	}{
		{
			name: "review approved",
			raw: `{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{
				"review":{"state":"approved","html_url":"https://r/1"},
				"pull_request":{"number":3038}}}`,
			ok: true, verb: "approved", num: "#3038", url: "https://r/1",
		},
		{
			name: "review requesting changes",
			raw: `{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{
				"review":{"state":"changes_requested","html_url":"https://r/2"},
				"pull_request":{"number":2989}}}`,
			ok: true, verb: "changes", num: "#2989", url: "https://r/2",
		},
		{
			name: "pull request merged, not merely closed",
			raw: `{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{
				"action":"closed",
				"pull_request":{"number":85,"merged":true,"html_url":"https://p/85"}}}`,
			ok: true, verb: "merged", num: "#85", url: "https://p/85",
		},
		{
			name: "pull request closed unmerged",
			raw: `{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{
				"action":"closed",
				"pull_request":{"number":3004,"merged":false,"html_url":"https://p/3004"}}}`,
			ok: true, verb: "closed", num: "#3004", url: "https://p/3004",
		},
		{
			name: "pull request opened",
			raw: `{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{
				"action":"opened","pull_request":{"number":412,"html_url":"https://p/412"}}}`,
			ok: true, verb: "opened", num: "#412", url: "https://p/412",
		},
		{
			name: "became a collaborator somewhere",
			raw: `{"type":"MemberEvent","repo":{"name":"awesome-selfhosted/awesome-selfhosted-data"},
				"payload":{"action":"added","member":{"login":"Rabenherz112"}}}`,
			ok: true, verb: "joined", num: "",
			url: "https://github.com/awesome-selfhosted/awesome-selfhosted-data",
		},
		{
			// The actor of a MemberEvent is whoever did the adding, so this
			// same event type also covers "I added someone to my repo". That
			// is a different statement and does not belong on the card.
			name: "added somebody else as a collaborator",
			raw: `{"type":"MemberEvent","repo":{"name":"Rabenherz112/some-project"},
				"payload":{"action":"added","member":{"login":"someone-else"}}}`,
			ok: false,
		},
		{
			name: "a collaborator being removed is not worth a line",
			raw: `{"type":"MemberEvent","repo":{"name":"a/b"},
				"payload":{"action":"removed","member":{"login":"Rabenherz112"}}}`,
			ok: false,
		},
		{
			name: "a release that was published",
			raw: `{"type":"ReleaseEvent","repo":{"name":"a/b"},"payload":{
				"action":"published",
				"release":{"tag_name":"v1.4.0","html_url":"https://r/v1.4.0"}}}`,
			ok: true, verb: "released", num: "v1.4.0", url: "https://r/v1.4.0",
		},
		{
			name: "issue closed",
			raw: `{"type":"IssuesEvent","repo":{"name":"a/b"},"payload":{
				"action":"closed","issue":{"number":7,"html_url":"https://i/7"}}}`,
			ok: true, verb: "closed", num: "#7", url: "https://i/7",
		},
		{
			name: "a push is not worth a line",
			raw:  `{"type":"PushEvent","repo":{"name":"a/b"},"payload":{}}`,
			ok:   false,
		},
		{
			name: "a dismissed review state we do not render",
			raw: `{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{
				"review":{"state":"dismissed"},"pull_request":{"number":1}}}`,
			ok: false,
		},
		{
			name: "a review event missing its payload is dropped, not panicked on",
			raw:  `{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{}}`,
			ok:   false,
		},
		{
			name: "a comment on an issue is not shown at all",
			raw: `{"type":"IssueCommentEvent","repo":{"name":"a/b"},"payload":{
				"action":"created","issue":{"number":9},"comment":{"html_url":"https://i/9#c"}}}`,
			ok: false,
		},
		{
			name: "a review left as a plain comment says nothing about the outcome",
			raw: `{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{
				"review":{"state":"commented","html_url":"https://r/9"},
				"pull_request":{"number":9}}}`,
			ok: false,
		},
		{
			name: "a commit comment is not shown either",
			raw: `{"type":"CommitCommentEvent","repo":{"name":"a/b"},"payload":{
				"action":"created","comment":{"html_url":"https://c/1"}}}`,
			ok: false,
		},
		{
			name: "a release that was only drafted or edited is not activity",
			raw: `{"type":"ReleaseEvent","repo":{"name":"a/b"},"payload":{
				"action":"edited","release":{"tag_name":"v1.4.0"}}}`,
			ok: false,
		},
		{
			name: "a starred repository is noise",
			raw:  `{"type":"WatchEvent","repo":{"name":"a/b"},"payload":{"action":"started"}}`,
			ok:   false,
		},
		{
			name: "a fork is noise",
			raw:  `{"type":"ForkEvent","repo":{"name":"a/b"},"payload":{}}`,
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, ok := convertEvent(decodeEvent(t, tc.raw), feedOwner)
			if ok != tc.ok {
				t.Fatalf("kept = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if ev.Verb != tc.verb {
				t.Errorf("verb = %q, want %q", ev.Verb, tc.verb)
			}
			if ev.Num != tc.num {
				t.Errorf("num = %q, want %q", ev.Num, tc.num)
			}
			if ev.URL != tc.url {
				t.Errorf("url = %q, want %q", ev.URL, tc.url)
			}
		})
	}
}

func TestRedactStripsCredentials(t *testing.T) {
	const in = "https://api.steampowered.com/x/?key=SECRET&steamid=1"
	got := redact(in)
	if got != "https://api.steampowered.com/x/?..." {
		t.Errorf("redact = %q", got)
	}
	if got == in {
		t.Error("redact left the query string in place")
	}
}

// TestNoCommentsSurviveConversion is a blanket guard rather than a case-by-case
// one. Comments are the easiest events to generate and the least informative to
// read, and GitHub has several kinds; a new one should not be able to appear on
// the card because nobody thought to exclude it.
func TestNoCommentsSurviveConversion(t *testing.T) {
	commentEvents := []string{
		`{"type":"IssueCommentEvent","repo":{"name":"a/b"},"payload":{"action":"created","issue":{"number":1},"comment":{"html_url":"https://x"}}}`,
		`{"type":"CommitCommentEvent","repo":{"name":"a/b"},"payload":{"comment":{"html_url":"https://x"}}}`,
		`{"type":"PullRequestReviewCommentEvent","repo":{"name":"a/b"},"payload":{"action":"created","pull_request":{"number":1},"comment":{"html_url":"https://x"}}}`,
		`{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{"review":{"state":"commented"},"pull_request":{"number":1}}}`,
		`{"type":"GollumEvent","repo":{"name":"a/b"},"payload":{}}`,
	}
	for _, raw := range commentEvents {
		ev, ok := convertEvent(decodeEvent(t, raw), feedOwner)
		if ok {
			t.Errorf("a comment reached the card as %q: %s", ev.Verb, raw)
		}
	}
}

// TestOnlyTheAgreedVerbs pins the set of verbs the card can ever show. Adding
// one is a deliberate act, not something that falls out of handling a new
// event type.
func TestOnlyTheAgreedVerbs(t *testing.T) {
	allowed := map[string]bool{
		"opened": true, "closed": true, "merged": true,
		"approved": true, "changes": true, "released": true, "joined": true,
	}

	raws := []string{
		`{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{"review":{"state":"approved"},"pull_request":{"number":1}}}`,
		`{"type":"PullRequestReviewEvent","repo":{"name":"a/b"},"payload":{"review":{"state":"changes_requested"},"pull_request":{"number":1}}}`,
		`{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{"action":"opened","pull_request":{"number":1}}}`,
		`{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{"action":"closed","pull_request":{"number":1,"merged":true}}}`,
		`{"type":"PullRequestEvent","repo":{"name":"a/b"},"payload":{"action":"closed","pull_request":{"number":1}}}`,
		`{"type":"IssuesEvent","repo":{"name":"a/b"},"payload":{"action":"opened","issue":{"number":1}}}`,
		`{"type":"IssuesEvent","repo":{"name":"a/b"},"payload":{"action":"closed","issue":{"number":1}}}`,
		`{"type":"ReleaseEvent","repo":{"name":"a/b"},"payload":{"action":"published","release":{"tag_name":"v1"}}}`,
		`{"type":"MemberEvent","repo":{"name":"a/b"},"payload":{"action":"added","member":{"login":"Rabenherz112"}}}`,
	}
	seen := map[string]bool{}
	for _, raw := range raws {
		if ev, ok := convertEvent(decodeEvent(t, raw), feedOwner); ok {
			if !allowed[ev.Verb] {
				t.Errorf("unexpected verb %q from %s", ev.Verb, raw)
			}
			seen[ev.Verb] = true
		}
	}
	for verb := range allowed {
		if !seen[verb] {
			t.Errorf("no event produced the verb %q, so it is unreachable", verb)
		}
	}
}
