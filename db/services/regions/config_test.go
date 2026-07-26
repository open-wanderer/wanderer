package regions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegionCatalog(t *testing.T) {
	t.Run("unset env var returns empty catalog, no error", func(t *testing.T) {
		t.Setenv("REGION_CATALOG_CONFIG_PATH", "")

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 0 {
			t.Fatalf("got %d regions, want 0", len(regions))
		}
	})

	t.Run("missing file returns empty catalog, no error", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("REGION_CATALOG_CONFIG_PATH", filepath.Join(dir, "does-not-exist.json"))

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 0 {
			t.Fatalf("got %d regions, want 0", len(regions))
		}
	})

	t.Run("empty file returns empty catalog, no error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "region_catalog.json")
		writeFile(t, path, "")
		t.Setenv("REGION_CATALOG_CONFIG_PATH", path)

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 0 {
			t.Fatalf("got %d regions, want 0", len(regions))
		}
	})

	t.Run("valid JSON of 2 well-formed regions returns both parsed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "region_catalog.json")
		writeFile(t, path, `[
			{"id": "bavaria", "name": "Bavaria", "bbox": [10.0, 47.0, 13.0, 50.0]},
			{"id": "tyrol", "name": "Tyrol", "bbox": [10.5, 46.5, 12.5, 47.8]}
		]`)
		t.Setenv("REGION_CATALOG_CONFIG_PATH", path)

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 2 {
			t.Fatalf("got %d regions, want 2", len(regions))
		}

		if regions[0].ID != "bavaria" || regions[0].Name != "Bavaria" {
			t.Fatalf("unexpected first region: %+v", regions[0])
		}
		wantBbox := [4]float64{10.0, 47.0, 13.0, 50.0}
		if regions[0].Bbox != wantBbox {
			t.Fatalf("got bbox %v, want %v", regions[0].Bbox, wantBbox)
		}

		if regions[1].ID != "tyrol" {
			t.Fatalf("unexpected second region: %+v", regions[1])
		}
	})

	t.Run("malformed JSON returns a non-nil error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "region_catalog.json")
		writeFile(t, path, `{not valid json`)
		t.Setenv("REGION_CATALOG_CONFIG_PATH", path)

		_, err := LoadRegionCatalog()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid id is skipped, sibling valid entry still returned", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "region_catalog.json")
		writeFile(t, path, `[
			{"id": "", "name": "Empty Id", "bbox": [10.0, 47.0, 13.0, 50.0]},
			{"id": "Bad Id", "name": "Bad Id", "bbox": [10.0, 47.0, 13.0, 50.0]},
			{"id": "../etc", "name": "Traversal", "bbox": [10.0, 47.0, 13.0, 50.0]},
			{"id": "a/b", "name": "Slash", "bbox": [10.0, 47.0, 13.0, 50.0]},
			{"id": "bavaria", "name": "Bavaria", "bbox": [10.0, 47.0, 13.0, 50.0]}
		]`)
		t.Setenv("REGION_CATALOG_CONFIG_PATH", path)

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 1 {
			t.Fatalf("got %d regions, want 1: %+v", len(regions), regions)
		}
		if regions[0].ID != "bavaria" {
			t.Fatalf("got id %q, want bavaria", regions[0].ID)
		}
	})

	t.Run("out-of-range bbox is skipped, sibling valid entry still returned", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "region_catalog.json")
		writeFile(t, path, `[
			{"id": "bad-lat", "name": "Bad Lat", "bbox": [10.0, -95.0, 13.0, 50.0]},
			{"id": "bad-lon", "name": "Bad Lon", "bbox": [-190.0, 47.0, 13.0, 50.0]},
			{"id": "bad-order-lon", "name": "Bad Order Lon", "bbox": [13.0, 47.0, 10.0, 50.0]},
			{"id": "bad-order-lat", "name": "Bad Order Lat", "bbox": [10.0, 50.0, 13.0, 47.0]},
			{"id": "bavaria", "name": "Bavaria", "bbox": [10.0, 47.0, 13.0, 50.0]}
		]`)
		t.Setenv("REGION_CATALOG_CONFIG_PATH", path)

		regions, err := LoadRegionCatalog()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(regions) != 1 {
			t.Fatalf("got %d regions, want 1: %+v", len(regions), regions)
		}
		if regions[0].ID != "bavaria" {
			t.Fatalf("got id %q, want bavaria", regions[0].ID)
		}
	})
}

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

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}
