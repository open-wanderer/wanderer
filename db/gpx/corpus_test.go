package gpx

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The corpus is one on-disk, language-neutral fixture set at the repository
// root, shared with the TypeScript and Dart suites. go test runs with the
// package directory as its working directory, so it is two levels up. The
// corpus is never copied or embedded. See fixtures/gpx-corpus/README.md for
// the contract this file implements.
var corpusRoot = filepath.Join("..", "..", "fixtures", "gpx-corpus")

// Absolute tolerances, declared once here and never inline in a test body.
// They are the README's tolerance table, with no Go-specific loosening.
const (
	metreTolerance  = 1e-6 // metres: distance, elevation gain and loss
	degreeTolerance = 1e-9 // degrees: centroid only
)

// corpusMinFixtures makes a corpus that silently shrinks fail loudly.
const corpusMinFixtures = 19

// parseLimit mirrors the web DEFAULT_GPX_FIELD_LIMIT (25 MiB).
const parseLimit = 25 << 20

type corpusWaypoint struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
}

type corpusExpected struct {
	ID         string   `json:"id"`
	Covers     []string `json:"covers"`
	Derivation string   `json:"derivation"`
	Notes      string   `json:"notes"`
	Metrics    struct {
		Distance      float64 `json:"distance"`
		ElevationGain float64 `json:"elevationGain"`
		ElevationLoss float64 `json:"elevationLoss"`
		DurationMs    int64   `json:"durationMs"`
		PointCount    int     `json:"pointCount"`
		BoundingBox   struct {
			MinLat float64 `json:"minLat"`
			MaxLat float64 `json:"maxLat"`
			MinLon float64 `json:"minLon"`
			MaxLon float64 `json:"maxLon"`
		} `json:"boundingBox"`
		Centroid struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"centroid"`
	} `json:"metrics"`
	Trail struct {
		Name          string           `json:"name"`
		Description   *string          `json:"description"`
		Lat           float64          `json:"lat"`
		Lon           float64          `json:"lon"`
		Date          *string          `json:"date"`
		Distance      float64          `json:"distance"`
		ElevationGain float64          `json:"elevationGain"`
		ElevationLoss float64          `json:"elevationLoss"`
		Duration      float64          `json:"duration"`
		MinLat        float64          `json:"minLat"`
		MaxLat        float64          `json:"maxLat"`
		MinLon        float64          `json:"minLon"`
		MaxLon        float64          `json:"maxLon"`
		Waypoints     []corpusWaypoint `json:"waypoints"`
	} `json:"trail"`
}

type corpusFixture struct {
	dir      string
	input    []byte
	expected corpusExpected
}

// loadCorpus reads every fixture directory under the corpus root.
func loadCorpus(t *testing.T) []corpusFixture {
	t.Helper()
	if _, err := os.Stat(corpusRoot); err != nil {
		t.Fatalf("GPX corpus not found at %q (%v): run go test with the working directory set to db/gpx", corpusRoot, err)
	}
	entries, err := os.ReadDir(corpusRoot) // already sorted by name
	if err != nil {
		t.Fatalf("reading corpus %q: %v", corpusRoot, err)
	}
	var fixtures []corpusFixture
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		input, err := os.ReadFile(filepath.Join(corpusRoot, e.Name(), "input.gpx"))
		if err != nil {
			t.Fatalf("fixture %s: %v", e.Name(), err)
		}
		raw, err := os.ReadFile(filepath.Join(corpusRoot, e.Name(), "expected.json"))
		if err != nil {
			t.Fatalf("fixture %s: %v", e.Name(), err)
		}
		var exp corpusExpected
		if err := json.Unmarshal(raw, &exp); err != nil {
			t.Fatalf("fixture %s: expected.json: %v", e.Name(), err)
		}
		fixtures = append(fixtures, corpusFixture{dir: e.Name(), input: input, expected: exp})
	}
	if len(fixtures) < corpusMinFixtures {
		t.Fatalf("corpus holds %d fixtures, want at least %d (a fixture was deleted or the path is wrong)", len(fixtures), corpusMinFixtures)
	}
	return fixtures
}

// assertCorpusMatch is the one comparison helper. A positive tol compares by
// absolute difference, anything else compares exactly.
func assertCorpusMatch(t *testing.T, field string, got, want, tol float64) {
	t.Helper()
	if tol > 0 {
		if diff := math.Abs(got - want); !(diff <= tol) {
			t.Errorf("field %q: %v not within %v of %v", field, got, tol, want)
		}
		return
	}
	if got != want {
		t.Errorf("field %q: got %v, want %v", field, got, want)
	}
}

func TestCorpusMetrics(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.dir, func(t *testing.T) {
			res, err := Parse(bytes.NewReader(f.input), parseLimit, "")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			want := f.expected.Metrics
			assertCorpusMatch(t, "distance", res.Distance, want.Distance, metreTolerance)
			assertCorpusMatch(t, "elevationGain", res.ElevationGain, want.ElevationGain, metreTolerance)
			assertCorpusMatch(t, "elevationLoss", res.ElevationLoss, want.ElevationLoss, metreTolerance)
			assertCorpusMatch(t, "durationMs", float64(res.DurationMs), float64(want.DurationMs), 0)
			assertCorpusMatch(t, "pointCount", float64(res.PointCount), float64(want.PointCount), 0)
			assertCorpusMatch(t, "boundingBox.minLat", res.Bounds.MinLat, want.BoundingBox.MinLat, 0)
			assertCorpusMatch(t, "boundingBox.maxLat", res.Bounds.MaxLat, want.BoundingBox.MaxLat, 0)
			assertCorpusMatch(t, "boundingBox.minLon", res.Bounds.MinLon, want.BoundingBox.MinLon, 0)
			assertCorpusMatch(t, "boundingBox.maxLon", res.Bounds.MaxLon, want.BoundingBox.MaxLon, 0)
			assertCorpusMatch(t, "centroid.lat", res.Centroid.Lat, want.Centroid.Lat, degreeTolerance)
			assertCorpusMatch(t, "centroid.lon", res.Centroid.Lon, want.Centroid.Lon, degreeTolerance)
		})
	}
}

func TestCorpusIntegrity(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.dir, func(t *testing.T) {
			if f.expected.ID != f.dir {
				t.Errorf("expected.id %q does not equal directory name %q", f.expected.ID, f.dir)
			}
			if f.expected.Derivation == "hand" {
				if _, err := os.Stat(filepath.Join(corpusRoot, f.dir, "DERIVATION.md")); err != nil {
					t.Errorf("derivation \"hand\" requires DERIVATION.md: %v", err)
				}
			}
		})
	}
}
