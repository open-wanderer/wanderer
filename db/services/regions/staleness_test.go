package regions

import (
	"strings"
	"testing"
)

func TestNeedsVectorRebuild(t *testing.T) {
	cases := []struct {
		name       string
		storedDate string
		currDate   string
		want       bool
	}{
		{"date advanced -> stale", "20260720", "20260721", true},
		{"same date -> fresh", "20260721", "20260721", false},
		{"never built (empty stored date) -> must build", "", "20260721", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := needsVectorRebuild(c.storedDate, c.currDate); got != c.want {
				t.Errorf("needsVectorRebuild(%q, %q) = %v, want %v", c.storedDate, c.currDate, got, c.want)
			}
		})
	}
}

func TestBboxChanged(t *testing.T) {
	configBbox := [4]float64{10.0, 47.0, 13.0, 50.0}

	t.Run("identical bbox -> false", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.0, 50.0); got != false {
			t.Errorf("bboxChanged = %v, want false", got)
		}
	})

	t.Run("min_lon differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.5, 47.0, 13.0, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("min_lat differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.5, 13.0, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("max_lon differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.5, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("max_lat differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.0, 50.5); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})
}

// TestValidProtomapsDateAndURLFormat asserts only the format contract, which
// holds regardless of whether the today or yesterday branch runs (network
// access is not mocked here — see staleness.go's doc comment).
func TestValidProtomapsDateAndURLFormat(t *testing.T) {
	date, url := ValidProtomapsDateAndURL()

	if len(date) != 8 {
		t.Fatalf("date %q has length %d, want 8", date, len(date))
	}
	for _, r := range date {
		if r < '0' || r > '9' {
			t.Fatalf("date %q contains non-digit character %q", date, r)
		}
	}

	if !strings.HasPrefix(url, "https://build.protomaps.com/") {
		t.Fatalf("url %q does not have expected prefix", url)
	}
	if !strings.HasSuffix(url, date+".pmtiles") {
		t.Fatalf("url %q does not have suffix %q", url, date+".pmtiles")
	}
}
