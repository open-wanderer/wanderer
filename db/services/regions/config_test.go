package regions

import (
	"path/filepath"
	"testing"
)

func TestIsValidRegionID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"bavaria", true},
		{"../etc", false},
		{"", false},
		{"A", false},
		{"a/b", false},
		{"algeria.algeria_central", true},
		{"people's_republic_of_china", true},
		{"a1_b-2.c'd", true},
		{"de-nrw", true},
		{"..", false},
		{"algeria..central", false},
		{"a\\b", false},
		{".hidden", false},
	}

	for _, c := range cases {
		if got := IsValidRegionID(c.id); got != c.want {
			t.Errorf("IsValidRegionID(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

// TestRegionIDCannotProduceTraversalPath asserts a malicious region id can
// never reach filepath.Join because IsValidRegionID rejects it first.
func TestRegionIDCannotProduceTraversalPath(t *testing.T) {
	maliciousIDs := []string{"../etc", "a/b", "../../pb_data", "..", "a\\b", "algeria..central", ".hidden"}

	for _, id := range maliciousIDs {
		if IsValidRegionID(id) {
			t.Errorf("IsValidRegionID(%q) = true, want false (path-traversal risk)", id)
		}
	}
}

func TestRegionArchivePath(t *testing.T) {
	got := RegionArchivePath("bavaria")
	want := filepath.Join(RegionCacheDir, "bavaria", "vector.pmtiles")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRegionDemPath(t *testing.T) {
	got := RegionDemPath("bavaria")
	want := filepath.Join(RegionCacheDir, "bavaria", "dem.pmtiles")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
