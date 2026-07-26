package commands

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// defaultCommitHash is the CoMaps (comaps/comaps) `main` HEAD commit resolved
// at implementation time (2026-07-25) via Codeberg's Forgejo commits API
// (GET https://codeberg.org/api/v1/repos/comaps/comaps/commits?limit=1&sha=main).
// A concrete SHA is baked in — never the literal "main" — so a re-run of this
// tool without --commit reproduces the exact same output (28-RESEARCH.md
// Open Question 3). The same commit is fetched from GitHub's mirror (see
// baseURL below) rather than Codeberg directly — Codeberg's raw-file
// endpoint enforces a ~250-requests/600s quota that a maintainer run
// against ~1,150 leaf .poly files routinely exhausts, turning a single run
// into repeated multi-minute stalls; comaps/comaps is mirrored to GitHub
// (github.com/comaps/comaps), and raw.githubusercontent.com serves
// identical byte-for-byte content for the same commit with no observed
// rate limiting for this access pattern.
const defaultCommitHash = "2528fbb91977201cf6d16b1b01ebf27eea342e85"

// commitHashPattern is the allow-list a --commit value must satisfy before it
// is interpolated into the fetch URL (Threat T-28-03).
var commitHashPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// SeedRow is one flattened row of the JSON array seed_regions.go writes:
// one entry per CoMaps hierarchy.txt node (group or leaf). Field tags match
// byte-for-byte what plan 28-03's migration reader struct expects. Polygon
// and Bbox are omitempty so group rows serialize without those keys
// (CATALOG-02: geometry is leaf-only).
type SeedRow struct {
	ComapsID       string         `json:"comaps_id"`
	ParentComapsID string         `json:"parent_comaps_id"`
	Path           string         `json:"path"`
	Depth          int            `json:"depth"`
	SortOrder      int            `json:"sort_order"`
	Name           string         `json:"name"`
	Kind           string         `json:"kind"`
	Polygon        map[string]any `json:"polygon,omitempty"`
	Bbox           []float64      `json:"bbox,omitempty"`
}

// SeedRegions returns the "seed-regions" Cobra command: a maintainer-run,
// dev-time-only tool that fetches CoMaps' hierarchy.txt and every leaf
// .poly file fresh from comaps/comaps's GitHub mirror at a pinned commit
// (D-01 — nothing raw is vendored into this repo, only this tool's
// flattened JSON output is ever committed), converts them via the plan
// 28-01 parsers (ParseHierarchy, ParsePoly), and writes the result to
// --out. The commit to fetch is a CLI flag with a baked-in default (D-02)
// so a maintainer can refresh the catalog ad hoc without editing source.
//
// The output is a gzip-compressed compact JSON seed (not pretty-printed):
// indenting the marshaled output inflates the full ~1,150-leaf catalog to
// ~730 MB, roughly 7x over GitHub's 100 MB hard per-file push limit —
// compact encoding alone still leaves ~216 MB, so the write path also
// gzip-compresses (default level 6) down to ~57 MB, comfortably
// distributable via a normal git push (28-04 gap closure, SEED-02).
//
// Unlike Dedup, this command's constructor takes no *pocketbase.PocketBase —
// it never touches a live database, only fetches, transforms, and writes a
// file.
func SeedRegions() *cobra.Command {
	var commit string
	var out string
	var limit int

	cmd := &cobra.Command{
		Use:   "seed-regions",
		Short: "Fetch CoMaps hierarchy.txt + .poly files from comaps/comaps's GitHub mirror and write a gzip-compressed flattened regions JSON seed",
		Run: func(cmd *cobra.Command, args []string) {
			if !commitHashPattern.MatchString(commit) {
				log.Fatalf("seed-regions: --commit %q is not a valid git commit hash (expected 7-40 hex characters)", commit)
			}

			baseURL := fmt.Sprintf("https://raw.githubusercontent.com/comaps/comaps/%s/data/", commit)

			hierarchyData, err := fetch(baseURL+"hierarchy.txt", 8<<20)
			if err != nil {
				log.Fatalf("seed-regions: failed to fetch hierarchy.txt: %v", err)
			}

			nodes, err := ParseHierarchy(hierarchyData)
			if err != nil {
				log.Fatalf("seed-regions: failed to parse hierarchy.txt: %v", err)
			}

			rows := make([]SeedRow, len(nodes))
			for i, n := range nodes {
				rows[i] = SeedRow{
					ComapsID:       n.ComapsID,
					ParentComapsID: n.ParentComapsID,
					Path:           n.Path,
					Depth:          n.Depth,
					SortOrder:      n.SortOrder,
					Name:           n.Name,
					Kind:           n.Kind,
				}
			}

			var groupCount, leafCount, fetchedLeaves int
			for i := range rows {
				if rows[i].Kind != "leaf" {
					groupCount++
					continue
				}
				leafCount++

				if limit > 0 && fetchedLeaves >= limit {
					continue
				}

				polyURL := baseURL + "borders/" + url.PathEscape(rows[i].ComapsID) + ".poly"
				polyData, err := fetch(polyURL, 32<<20)
				if err != nil {
					log.Fatalf("seed-regions: failed to fetch .poly for region %q: %v", rows[i].ComapsID, err)
				}

				geometry, bbox, err := ParsePoly(polyData)
				if err != nil {
					log.Fatalf("seed-regions: failed to parse .poly for region %q: %v", rows[i].ComapsID, err)
				}

				rows[i].Polygon = geometry
				rows[i].Bbox = []float64{bbox[0], bbox[1], bbox[2], bbox[3]}
				fetchedLeaves++
			}

			data, err := json.Marshal(rows)
			if err != nil {
				log.Fatalf("seed-regions: failed to marshal seed JSON: %v", err)
			}

			outFile, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				log.Fatalf("seed-regions: failed to create %s: %v", out, err)
			}
			defer outFile.Close()

			gzWriter := gzip.NewWriter(outFile) // DefaultCompression (level 6) — the measured ~57MB point; do not drop below it
			if _, err := gzWriter.Write(data); err != nil {
				log.Fatalf("seed-regions: failed to write gzip data to %s: %v", out, err)
			}
			if err := gzWriter.Close(); err != nil {
				log.Fatalf("seed-regions: failed to flush gzip trailer to %s: %v", out, err)
			}

			fmt.Printf("seed-regions: wrote %d rows (%d groups, %d leaves, %d leaf .poly files fetched) to %s (gzip-compressed compact JSON)\n",
				len(rows), groupCount, leafCount, fetchedLeaves, out)
		},
	}

	cmd.Flags().StringVar(&commit, "commit", defaultCommitHash, "CoMaps (comaps/comaps) commit hash to fetch hierarchy.txt/.poly data from")
	cmd.Flags().StringVar(&out, "out", "migrations/initial_data/regions_seed.json.gz", "output path for the gzip-compressed flattened regions JSON seed")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit the number of leaf .poly files fetched (0 = all leaves); a dev smoke-test aid")

	return cmd
}

// maxFetchRetries bounds how many times fetch retries a single URL after a
// 429 (rate limited) response before giving up. Codeberg's raw-file
// endpoint enforces a tight per-window quota (observed: 250 requests /
// 600s) that a maintainer run against ~1,150 leaf .poly files routinely
// exhausts — this is a transient upstream condition, not malformed data,
// so it is retried rather than treated as a fatal parse/fetch error.
const maxFetchRetries = 10

// fetch performs an HTTP GET against rawURL and reads at most maxBytes of
// the response body, bounding memory use against an unexpectedly large
// upstream response (Threat T-28-04). A 429 response is retried up to
// maxFetchRetries times, sleeping for the duration named by the response's
// Retry-After header (or a conservative default if absent) between
// attempts. A non-200/429 status or a read error is returned as a
// descriptive error, never surfaced as a process abort by this function
// itself — callers decide whether a failure is fatal.
func fetch(rawURL string, maxBytes int64) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= maxFetchRetries; attempt++ {
		data, retryAfter, err := doFetch(rawURL, maxBytes)
		if err == nil {
			return data, nil
		}
		if retryAfter <= 0 {
			return nil, err
		}
		lastErr = err
		if attempt == maxFetchRetries {
			break
		}
		fmt.Printf("seed-regions: rate limited fetching %s, waiting %s before retry %d/%d\n", rawURL, retryAfter, attempt+1, maxFetchRetries)
		time.Sleep(retryAfter)
	}

	return nil, fmt.Errorf("GET %s: exceeded %d retries: %w", rawURL, maxFetchRetries, lastErr)
}

// doFetch is fetch's single-attempt HTTP GET. It returns a non-zero
// retryAfter duration only when the response was a 429, signaling to fetch
// that this attempt is retryable.
func doFetch(rawURL string, maxBytes int64) ([]byte, time.Duration, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("GET %s: rate limited (429)", rawURL)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("GET %s: unexpected status %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: read body: %w", rawURL, err)
	}

	return data, 0, nil
}

// parseRetryAfter interprets a Retry-After header value (seconds, per RFC
// 9110) with a conservative fallback when the header is missing or
// unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 30 * time.Second
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 30 * time.Second
}
