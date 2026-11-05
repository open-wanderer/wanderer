package regions

import (
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"
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

func TestArchivePathsFollowTheDataDir(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: filepath.Join("srv", "wanderer", "pb_data")})

	if got, want := CacheDir(app), filepath.Join("srv", "wanderer", "pb_data", "region_archives"); got != want {
		t.Errorf("CacheDir = %q, want %q", got, want)
	}
	if got, want := RegionArchivePath(app, "bavaria"), filepath.Join(CacheDir(app), "bavaria", "vector.pmtiles"); got != want {
		t.Errorf("RegionArchivePath = %q, want %q", got, want)
	}
	if got, want := RegionDemPath(app, "bavaria"), filepath.Join(CacheDir(app), "bavaria", "dem.pmtiles"); got != want {
		t.Errorf("RegionDemPath = %q, want %q", got, want)
	}
}

func TestRelativeInside(t *testing.T) {
	for _, test := range []struct {
		name, root, path string
		want             string
		wantOK           bool
	}{
		{"default location", "/pb_data", "/pb_data/region_archives", "region_archives", true},
		{"other name", "/pb_data", "/pb_data/offline", "offline", true},
		{"nested", "/pb_data", "/pb_data/cache/regions", "cache/regions", true},
		{"relative data dir", "pb_data", "pb_data/region_archives", "region_archives", true},
		{"outside", "/pb_data", "/mnt/archives", "", false},
		{"sibling sharing the prefix", "/pb_data", "/pb_data_archives", "", false},
		{"the data dir itself", "/pb_data", "/pb_data", "", false},
		{"parent of the data dir", "/data/pb_data", "/data", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := relativeInside(test.root, test.path)
			if got != test.want || ok != test.wantOK {
				t.Errorf("relativeInside(%q, %q) = %q, %v; want %q, %v", test.root, test.path, got, ok, test.want, test.wantOK)
			}
		})
	}
}
