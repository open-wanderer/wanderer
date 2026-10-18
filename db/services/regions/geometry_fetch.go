package regions

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// comapsGitHubRawBase and comapsCodebergRawBase are the two hardcoded
// upstream hosts a leaf's .poly is fetched from, in fallback order
// (no env var, no injectable seam). GitHub's mirror is primary:
// it observably imposes no rate-limit window for a single on-demand
// fetch, whereas Codeberg's Forgejo raw endpoint enforces a roughly
// 250-requests/600s quota (see db/commands/seed_regions.go's doc comment
// on why the maintainer seed generator also prefers it). That quota is
// irrelevant to a single leaf's on-demand fetch, but GitHub stays primary
// for consistency with the generator's own host preference. Codeberg
// remains CoMaps' canonical upstream home and is the correct fallback if
// the GitHub mirror is ever unreachable or falls behind.
const (
	comapsGitHubRawBase   = "https://raw.githubusercontent.com/comaps/comaps"
	comapsCodebergRawBase = "https://codeberg.org/comaps/comaps/raw/commit"
)

// maxPolyBytes bounds a single .poly response read, preserved verbatim
// from the seed-regions generator's own bound, so removing the committed
// seed does not also remove the response-size guard.
const maxPolyBytes = 32 << 20

// polyFetchAttempts and polyFetchRetryCap define this package's own,
// tighter retry budget for the on-demand build/route path, as
// opposed to seed_regions.go's patient maxFetchRetries = 10 with 30s
// Retry-After sleeps. This fetch runs synchronously inside a per-region
// cron build across roughly 100 enabled regions, and inline in an HTTP
// request handler — a flaky upstream must not be allowed to grind either
// for hours. Worst case per host is polyFetchAttempts attempts, each
// separated by at most polyFetchRetryCap: (polyFetchAttempts-1) *
// polyFetchRetryCap = 5s of sleep, across 2 hosts = 10s of intentional
// backoff, plus request/timeout time bounded by polyHTTPClient's 15s
// Timeout per attempt (2 attempts * 2 hosts * 15s = 60s worst case) —
// roughly 70-80s total per region in the fully-down case, versus tens of
// minutes under the generator's budget.
const (
	polyFetchAttempts = 2
	polyFetchRetryCap = 5 * time.Second
)

// polyHTTPClient is the bounded HTTP client used for every .poly fetch,
// matching db/federation/activity.go's bounded-client convention rather
// than the package-default http.Get.
var polyHTTPClient = &http.Client{Timeout: 15 * time.Second}

// catalogCommitPattern is the allow-list a commit SHA must satisfy before
// it is interpolated into a fetch URL, mirroring
// db/commands/seed_regions.go's commitHashPattern precedent. Declared
// locally rather than imported from package
// commands so this package incurs no services/regions -> commands
// dependency.
var catalogCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// comapsIDPattern is the allow-list a CoMaps leaf id must satisfy before
// it is interpolated into a fetch URL. It is grounded in
// the verified character set of the live 1306-row catalog: every real
// leaf id is drawn from letters, digits, underscore, hyphen and space,
// with a single apostrophe-bearing exception ("People's Republic of
// China") that is a kind=group row and therefore never reaches this
// function. The pattern structurally excludes '/', '\', '%' and '.', so
// a path-traversal sequence like ".." is unrepresentable, and
// url.PathEscape is additionally applied on top before URL construction.
var comapsIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _'-]*$`)

// PolySourceURLs validates commitSHA and comapsID against their allow-lists
// and returns the two .poly URLs to try, in fallback order: the GitHub
// mirror first, then CoMaps' canonical Codeberg repository. Kept
// pure and exported so the fallback ordering is unit-testable with no
// network access.
func PolySourceURLs(commitSHA, comapsID string) ([]string, error) {
	if !catalogCommitPattern.MatchString(commitSHA) {
		return nil, fmt.Errorf("geometry_fetch: invalid commit hash %q (expected 7-40 hex characters)", commitSHA)
	}
	if !comapsIDPattern.MatchString(comapsID) {
		return nil, fmt.Errorf("geometry_fetch: invalid comaps id %q", comapsID)
	}

	escapedID := url.PathEscape(comapsID)

	return []string{
		fmt.Sprintf("%s/%s/data/borders/%s.poly", comapsGitHubRawBase, commitSHA, escapedID),
		fmt.Sprintf("%s/%s/data/borders/%s.poly", comapsCodebergRawBase, commitSHA, escapedID),
	}, nil
}

// fetchPolyBytes performs a bounded, retried HTTP GET against a single
// .poly URL, following doFetch's structure in db/commands/seed_regions.go:
// a 429 is retried (up to polyFetchAttempts total attempts) after sleeping
// for the smaller of the parsed Retry-After and polyFetchRetryCap; any
// other non-200 status is a descriptive, non-retryable error. The response
// body is read through io.LimitReader(resp.Body, maxPolyBytes) to bound
// memory use against an unexpectedly large upstream response.
func fetchPolyBytes(rawURL string) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= polyFetchAttempts; attempt++ {
		data, retryAfter, err := doFetchPoly(rawURL)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if retryAfter <= 0 {
			return nil, err
		}
		if attempt == polyFetchAttempts {
			break
		}
		if retryAfter > polyFetchRetryCap {
			retryAfter = polyFetchRetryCap
		}
		time.Sleep(retryAfter)
	}

	return nil, fmt.Errorf("GET %s: exceeded %d attempts: %w", rawURL, polyFetchAttempts, lastErr)
}

// doFetchPoly is fetchPolyBytes's single-attempt HTTP GET. It returns a
// non-zero retryAfter duration only when the response was a 429,
// signaling to fetchPolyBytes that this attempt is retryable.
func doFetchPoly(rawURL string) ([]byte, time.Duration, error) {
	resp, err := polyHTTPClient.Get(rawURL)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, parsePolyRetryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("GET %s: rate limited (429)", rawURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("GET %s: unexpected status %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPolyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: read body: %w", rawURL, err)
	}

	return data, 0, nil
}

// parsePolyRetryAfter interprets a Retry-After header value (seconds, per
// RFC 9110), falling back to polyFetchRetryCap when the header is missing
// or unparseable — deliberately shorter than seed_regions.go's 30-second
// fallback, matching this path's tighter budget.
func parsePolyRetryAfter(v string) time.Duration {
	if v == "" {
		return polyFetchRetryCap
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return polyFetchRetryCap
}

// describePolyFetchFailure composes an error naming every upstream URL
// that was attempted, wrapping lastErr with %w so callers can still
// errors.Is/As through it. Kept pure so the two-upstream failure message
// is assertable in a unit test with no network access.
func describePolyFetchFailure(urls []string, lastErr error) error {
	return fmt.Errorf("geometry_fetch: failed to fetch .poly from all upstreams %v: %w", urls, lastErr)
}

// FetchGeometry fetches a CoMaps leaf's .poly at commitSHA, trying the
// GitHub mirror first and CoMaps' canonical Codeberg repository second
//, and converts the first successfully-fetched response into a
// GeoJSON geometry plus bbox via ParsePoly — the single canonical parser
// also used by the retired seed generator (the equivalence bar).
//
// A validation error from PolySourceURLs is returned unchanged. If every
// URL fails to fetch, the returned error names every upstream that was
// tried. If ParsePoly itself fails on bytes
// that were successfully fetched, that parse error is returned wrapped
// with comapsID and the process does not fall through to the other
// host — a parse failure means the fetched content is malformed, not that
// the host is unavailable, so retrying against a different host would not
// help.
func FetchGeometry(commitSHA, comapsID string) (map[string]any, [4]float64, error) {
	urls, err := PolySourceURLs(commitSHA, comapsID)
	if err != nil {
		return nil, [4]float64{}, err
	}

	var lastErr error
	for _, u := range urls {
		data, err := fetchPolyBytes(u)
		if err != nil {
			lastErr = err
			continue
		}

		geometry, bbox, err := ParsePoly(data)
		if err != nil {
			return nil, [4]float64{}, fmt.Errorf("geometry_fetch: parse .poly for %s: %w", comapsID, err)
		}
		return geometry, bbox, nil
	}

	return nil, [4]float64{}, describePolyFetchFailure(urls, lastErr)
}
