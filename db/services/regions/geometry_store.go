package regions

import (
	"fmt"
	"log"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ResolveGeometry is the single, shared read-through entry point for a
// leaf region's geometry (GeoJSON polygon + bbox), used by both
// buildRegion (build path) and plan 32-05's HTTP route (request path).
//
// Persistence is derived internally from the region record's own enabled
// flag — there is deliberately no `persist` parameter, no opt-in query
// flag, and no caller-supplied override anywhere in this phase. A
// caller cannot induce a write for a region that is not enabled.
//
// Build-path callers (buildRegion, via BuildAllLocked) always satisfy the
// persist condition because BuildAllLocked only ever iterates enabled
// leaves — so the refetch below always writes on that path. The HTTP
// route's disabled-region hover flow deliberately does not: hovering a
// disabled region is a pass-through with no write.
func ResolveGeometry(app core.App, region *core.Record) (map[string]any, [4]float64, error) {
	path := region.GetString("path")
	if !IsValidRegionID(path) {
		return nil, [4]float64{}, fmt.Errorf("geometry_store: invalid region path %q", path)
	}

	comapsID := region.GetString("comaps_id")
	if comapsID == "" {
		return nil, [4]float64{}, fmt.Errorf("geometry_store: region %s has no comaps_id", path)
	}

	commitSHA := region.GetString("catalog_commit")
	if commitSHA == "" {
		return nil, [4]float64{}, fmt.Errorf("geometry_store: region %s has no pinned commit SHA (catalog predates the plan 32-02 migration)", path)
	}

	persist := region.GetBool("enabled")

	// Resolution order: read the cached row, then decide whether it is
	// usable. An absent row and a malformed row are treated identically —
	// both fall through to the refetch below. This is the trap the phase
	// exists to close: a literal "fetch on cache miss" implementation would
	// silently skip the malformed case, because a corrupt row *exists* so
	// there is no miss, leaving the region permanently broken (the old
	// builder.go:238-241 dead end this replaces).
	cached, lookupErr := app.FindFirstRecordByFilter("region_geometry",
		"path = {:path}", dbx.Params{"path": path})

	if lookupErr == nil {
		var polygon map[string]any
		var bboxSlice []float64
		polyErr := cached.UnmarshalJSONField("polygon", &polygon)
		bboxErr := cached.UnmarshalJSONField("bbox", &bboxSlice)

		if polyErr == nil && bboxErr == nil && len(polygon) > 0 && len(bboxSlice) == 4 {
			bbox := [4]float64{bboxSlice[0], bboxSlice[1], bboxSlice[2], bboxSlice[3]}
			return polygon, bbox, nil
		}
	}

	// Absent row (lookupErr != nil) or malformed row (fell through above):
	// refetch from CoMaps. Only if this fails does the caller's setError
	// last-resort apply.
	geometry, bbox, err := FetchGeometry(commitSHA, comapsID)
	if err != nil {
		return nil, [4]float64{}, fmt.Errorf("geometry_store: resolve geometry for region %s: %w", path, err)
	}

	if persist {
		record := cached
		if lookupErr != nil || record == nil {
			collection, err := app.FindCollectionByNameOrId("region_geometry")
			if err != nil {
				log.Printf("[regions] geometry_store: region_geometry collection not found, skipping persist for %s: %v", path, err)
				return geometry, bbox, nil
			}
			record = core.NewRecord(collection)
			record.Set("path", path)
		}

		record.Set("polygon", geometry)
		record.Set("bbox", []float64{bbox[0], bbox[1], bbox[2], bbox[3]})

		if err := app.Save(record); err != nil {
			// A persist failure must be logged and swallowed, not returned:
			// the caller already has valid geometry in hand, and failing a
			// build because a cache write failed would be strictly worse
			// than proceeding.
			log.Printf("[regions] geometry_store: failed to persist region_geometry for %s: %v", path, err)
		}
	}

	return geometry, bbox, nil
}
