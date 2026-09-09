// Package sources fetches the live data behind the cards.
//
// Each source is independent and each one is allowed to fail. The generator
// runs unattended on a schedule and commits its output, so a source that is
// down must not fail the run or blank a card: it returns an error, the caller
// logs it, and that section of the snapshot keeps the values it already had.
package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UserAgent identifies the generator to the APIs it calls. GitHub rejects
// requests without one.
const UserAgent = "Rabenherz112-profile-cards/1 (+https://github.com/Rabenherz112/Rabenherz112)"

// DefaultTimeout bounds a single request. The workflow should finish quickly
// even when a self-hosted service is unreachable rather than hanging.
const DefaultTimeout = 20 * time.Second

// Client is the HTTP client every source shares.
var Client = &http.Client{Timeout: DefaultTimeout}

// maxBody caps how much of a response is read, so a misconfigured endpoint
// returning something huge cannot exhaust memory.
const maxBody = 8 << 20 // 8 MiB

// getJSON performs a GET and decodes the JSON body into out.
func getJSON(ctx context.Context, url string, header http.Header, out any) error {
	return doJSON(ctx, http.MethodGet, url, header, nil, out)
}

// postJSON performs a POST with a JSON body and decodes the JSON response.
func postJSON(ctx context.Context, url string, header http.Header, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return doJSON(ctx, http.MethodPost, url, header, b, out)
}

func doJSON(ctx context.Context, method, url string, header http.Header, body []byte, out any) error {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}

	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The snippet helps diagnose a bad endpoint from the workflow log. It
		// is deliberately short: error bodies can echo request details back.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("%s %s: %s: %s", method, redact(url), resp.Status,
			strings.TrimSpace(string(snippet)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out)
}

// getBytes downloads a small binary resource, such as a cover image.
func getBytes(ctx context.Context, url string, limit int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := Client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GET %s: %s", redact(url), resp.Status)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", fmt.Errorf("GET %s: unexpected content type %q", redact(url), mime)
	}
	return b, mime, nil
}

// redact strips the query string from a URL before it reaches a log, because
// some of these APIs take the key as a query parameter.
func redact(url string) string {
	if i := strings.IndexByte(url, '?'); i >= 0 {
		return url[:i] + "?..."
	}
	return url
}

// fetchedNow is the timestamp recorded against a section that has just
// refreshed. It is truncated to the second: these end up in a committed JSON
// file, and sub-second digits would only add noise to every diff.
func fetchedNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

// base returns the API root to use: the configured one when set, otherwise the
// public default. Any trailing slash is trimmed so callers can concatenate a
// path without doubling it up.
func base(configured, fallback string) string {
	if configured == "" {
		return strings.TrimRight(fallback, "/")
	}
	return strings.TrimRight(configured, "/")
}
