package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
// tool without --commit reproduces the exact same output. The same commit
// is fetched from GitHub's mirror (see
// baseURL below) rather than Codeberg directly — comaps/comaps is mirrored
// to GitHub (github.com/comaps/comaps), and raw.githubusercontent.com serves
// identical byte-for-byte content for the same commit with no observed rate
// limiting. With only hierarchy.txt now fetched, a run is a single
// request either host could serve, but the GitHub mirror stays primary for
// consistency with the commit's provenance.
const defaultCommitHash = "2528fbb91977201cf6d16b1b01ebf27eea342e85"

// commitHashPattern is the allow-list a --commit value must satisfy before it
// is interpolated into the fetch URL.
var commitHashPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// SeedRow is one flattened row of the JSON array seed_regions.go writes:
// one entry per CoMaps hierarchy.txt node (group or leaf). Field tags match
// byte-for-byte what the migration reader struct expects. It carries pure
// hierarchy only — no geometry: a leaf's polygon/bbox are
// fetched on demand at runtime instead of being carried in this catalog.
type SeedRow struct {
	ComapsID       string `json:"comaps_id"`
	ParentComapsID string `json:"parent_comaps_id"`
	Path           string `json:"path"`
	Depth          int    `json:"depth"`
	SortOrder      int    `json:"sort_order"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
}

// SeedCatalog is the top-level shape written to --out: the pinned commit
// this catalog was generated from, recorded once, plus every
// flattened hierarchy row. Storing the commit inside the artifact rather
// than as a separate shared Go const means a regeneration with a different
// --commit can never silently desync the recorded provenance from the rows
// that were actually fetched.
type SeedCatalog struct {
	Commit string    `json:"commit"`
	Rows   []SeedRow `json:"rows"`
}

// SeedRegions returns the "seed-regions" Cobra command: a maintainer-run,
// dev-time-only tool that fetches CoMaps' hierarchy.txt fresh from
// comaps/comaps's GitHub mirror at a pinned commit (nothing raw is
// vendored into this repo, only this tool's flattened JSON output is ever
// committed), converts it via ParseHierarchy, and writes the result to
// --out as a SeedCatalog value. The commit to fetch is a CLI flag with a
// baked-in default so a maintainer can refresh the catalog ad hoc
// without editing source.
//
// The run issues exactly one HTTP request: with geometry no longer
// carried in this catalog, there is nothing left to derive from a leaf's
// .poly file, so the ~1,150-file scrape this tool used to perform is gone.
// The output is plain, pretty-printed JSON — no gzip layer — since
// the pure-hierarchy catalog is small enough (~292 KB) to commit and review
// directly; a maintainer refresh is now a reviewable diff rather than an
// opaque compressed binary blob.
//
// Unlike Dedup, this command's constructor takes no *pocketbase.PocketBase —
// it never touches a live database, only fetches, transforms, and writes a
// file.
func SeedRegions() *cobra.Command {
	var commit string
	var out string

	cmd := &cobra.Command{
		Use:   "seed-regions",
		Short: "Fetch CoMaps hierarchy.txt from comaps/comaps's GitHub mirror and write a plain, pretty-printed, hierarchy-only regions JSON catalog",
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

			var groupCount, leafCount int
			for i := range rows {
				if rows[i].Kind == "leaf" {
					leafCount++
				} else {
					groupCount++
				}
			}

			catalog := SeedCatalog{Commit: commit, Rows: rows}

			data, err := json.MarshalIndent(catalog, "", "  ")
			if err != nil {
				log.Fatalf("seed-regions: failed to marshal seed JSON: %v", err)
			}

			if err := os.WriteFile(out, data, 0o644); err != nil {
				log.Fatalf("seed-regions: failed to write %s: %v", out, err)
			}

			fmt.Printf("seed-regions: wrote %d rows (%d groups, %d leaves) to %s (plain pretty-printed JSON, commit %s, one HTTP request)\n",
				len(rows), groupCount, leafCount, out, commit)
		},
	}

	cmd.Flags().StringVar(&commit, "commit", defaultCommitHash, "CoMaps (comaps/comaps) commit hash to fetch hierarchy.txt from")
	cmd.Flags().StringVar(&out, "out", "migrations/initial_data/regions_seed.json", "output path for the plain, pretty-printed, hierarchy-only regions JSON catalog")

	return cmd
}

// maxFetchRetries bounds how many times fetch retries a single URL after a
// 429 (rate limited) response before giving up. Codeberg's raw-file
// endpoint enforces a tight per-window quota (observed: 250 requests /
// 600s); this generator now issues only one request per run, but the
// patient retry budget is kept — the build path gets its own tighter
// budget rather than reusing this one.
const maxFetchRetries = 10

// fetch performs an HTTP GET against rawURL and reads at most maxBytes of
// the response body, bounding memory use against an unexpectedly large
// upstream response. A 429 response is retried up to
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
