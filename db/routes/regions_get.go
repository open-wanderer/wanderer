package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionsList returns the full group+leaf region catalog from the seeded
// `regions` table, joined with each leaf's build state from
// `region_archives`. The `regions` table is the source of truth for which
// regions exist.
//
// Every row (both kind=group and kind=leaf) is returned, regardless of
// `enabled`, so a client can render group node names/labels for the whole
// tree — trimming is a trivial later filter,
// but omitting group rows now would mean re-adding data later. Each entry
// always carries the hierarchy fields id/name/kind/parent/path/depth; leaf
// rows additionally carry enabled and the existing build-state shape
// (status/version/vector_url/vector_size/dem_status/dem_url/dem_size/error),
// joined to region_archives on region_archives.path == the leaf's path
// (path is the provably-unique seeded key, not the record's own opaque id).
//
// A leaf's bbox now comes from region_geometry, joined in one
// projected path+bbox query built once before the entry loop — never per
// leaf, and never selecting the polygon column, which would otherwise pull
// tens of megabytes of geometry into every catalog response. A leaf
// carries `bbox` only once its geometry has resolved,
// which under the persist-on-enable rule means in practice only for
// enabled regions: an unfiltered listing therefore returns disabled leaves
// with no `bbox` key at all, and the existing Flutter
// parser already drops entries missing required fields, which is why the
// pre-existing comment about group rows above applies here too. The
// `?enabled=true` filtered listing that the shipped app uses is unaffected,
// because an enabled region cannot exist without its geometry having
// resolved. An enabled region whose geometry fetch failed also has no bbox
// at all — its `status` is already "error"
// and its `error` field already names the upstreams tried, so the absence
// of `bbox` is explained by fields the response already contains.
//
// Auth is enforced at the route-group level in main.go (apis.RequireAuth()),
// so — unlike map_cells_id.go's unauthenticated /map/cells routes — every
// handler in this file (listing AND both download routes) requires a logged
// in user. An auth-gated listing with unauthenticated file bytes
// would defeat that requirement, so the download handlers below enforce the
// same posture at the group level too.
func RegionsList(e *core.RequestEvent) error {
	records, err := e.App.FindAllRecords("regions")
	if err != nil {
		return e.InternalServerError("failed to load region catalog", err)
	}

	// `?enabled=true` (opt-in) prunes the response to only enabled leaves plus
	// the ancestor groups needed to reach them — the filtering the Flutter
	// client used to do locally after downloading the whole catalog. Any other
	// value / no param returns the full catalog unchanged (backward-compat: the
	// dev harness and any admin tooling still expect every row).
	filtering := e.Request.URL.Query().Get("enabled") == "true"

	// The group rows to keep = ancestors of some enabled leaf. Derived from the
	// enabled leaves' materialized paths in one cheap pre-pass; a group whose
	// descendants are all disabled is never an ancestor, so it drops out.
	var groupPaths map[string]struct{}
	if filtering {
		leafPaths := make([]string, 0, len(records))
		for _, r := range records {
			if r.GetString("kind") == "leaf" && r.GetBool("enabled") {
				leafPaths = append(leafPaths, r.GetString("path"))
			}
		}
		groupPaths = regions.AncestorGroupPaths(leafPaths)
	}

	// Built once, before the entry loop, via a single projected query that
	// selects only path+bbox — never polygon, which would otherwise add
	// tens of megabytes to this response (see doc comment above). A query
	// error must not fail the whole request; log it and continue with an
	// empty map so the listing still serves hierarchy and build state.
	leafBboxes := make(map[string][]float64)
	rows, err := e.App.DB().Select("path", "bbox").From("region_geometry").Rows()
	if err != nil {
		log.Printf("[regions] failed to query region_geometry for bbox join: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var path, bboxJSON string
			if err := rows.Scan(&path, &bboxJSON); err != nil {
				log.Printf("[regions] failed to scan region_geometry row: %v", err)
				continue
			}
			var bbox []float64
			if err := json.Unmarshal([]byte(bboxJSON), &bbox); err != nil || len(bbox) != 4 {
				continue
			}
			leafBboxes[path] = bbox
		}
	}

	entries := make([]map[string]any, 0, len(records))
	for _, r := range records {
		if filtering {
			if r.GetString("kind") == "leaf" {
				if !r.GetBool("enabled") {
					continue
				}
			} else if _, ok := groupPaths[r.GetString("path")]; !ok {
				continue
			}
		}

		entry := map[string]any{
			"id":         r.Id,
			"name":       r.GetString("name"),
			"kind":       r.GetString("kind"),
			"parent":     r.GetString("parent"), // relation field's raw value = parent record id, "" for roots
			"path":       r.GetString("path"),
			"depth":      r.GetInt("depth"),
			"sort_order": r.GetInt("sort_order"), // sibling ordering hint, on every row (group and leaf)
		}

		if r.GetString("kind") != "leaf" {
			// Group rows are hierarchy-only — no bbox/build-state. The
			// existing Flutter parser (RegionCatalogEntry.fromJson) drops
			// entries missing its required bbox/status fields, so group
			// rows are ignored by the pre-Phase-31 app rather than crashing
			// it.
			entries = append(entries, entry)
			continue
		}

		path := r.GetString("path")

		if bbox, ok := leafBboxes[path]; ok {
			entry["bbox"] = bbox
		}
		entry["enabled"] = r.GetBool("enabled")

		archiveRecords, _ := e.App.FindAllRecords("region_archives",
			dbx.NewExp("path = {:path}", dbx.Params{"path": path}),
		)

		if len(archiveRecords) == 0 {
			// No build has ever started for this region yet — first build
			// pending.
			entry["status"] = "building"
			entries = append(entries, entry)
			continue
		}

		archive := archiveRecords[0]
		status := archive.GetString("status")
		entry["status"] = status

		if version := archive.GetString("vector_built_date"); version != "" {
			entry["version"] = version
		}

		if status == "ready" {
			if _, err := os.Stat(regions.RegionArchivePath(path)); err == nil {
				entry["vector_url"] = "/api/v1/regions/" + path + "/download"
				entry["vector_size"] = int64(archive.GetFloat("vector_size_bytes"))
			}
		}

		if status == "error" {
			entry["error"] = archive.GetString("error_message")
		}

		demStatus := archive.GetString("dem_status")
		if demStatus != "" {
			entry["dem_status"] = demStatus
		}
		if demStatus == "ready" {
			if _, err := os.Stat(regions.RegionDemPath(path)); err == nil {
				entry["dem_url"] = "/api/v1/regions/" + path + "/download-dem"
				entry["dem_size"] = int64(archive.GetFloat("dem_size_bytes"))
			}
		}

		entries = append(entries, entry)
	}

	return e.JSON(http.StatusOK, entries)
}

// RegionArchiveDownload streams a region's pre-built vector archive bytes.
// e.FileFS serves via the stdlib file server, which supports HTTP Range
// requests — the resumable download engine relies on this.
func RegionArchiveDownload(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if !regions.IsValidRegionID(id) {
		return e.BadRequestError("invalid region id", nil)
	}

	if _, err := os.Stat(regions.RegionArchivePath(id)); err != nil {
		return e.NotFoundError("Region archive not ready yet", nil)
	}

	return e.FileFS(os.DirFS(regions.RegionCacheDir), filepath.Join(id, "vector.pmtiles"))
}

// RegionArchiveDownloadDem streams a region's pre-built DEM archive bytes.
// Same Range-support note as RegionArchiveDownload applies here.
func RegionArchiveDownloadDem(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if !regions.IsValidRegionID(id) {
		return e.BadRequestError("invalid region id", nil)
	}

	if _, err := os.Stat(regions.RegionDemPath(id)); err != nil {
		return e.NotFoundError("Region DEM archive not ready yet", nil)
	}

	return e.FileFS(os.DirFS(regions.RegionCacheDir), filepath.Join(id, "dem.pmtiles"))
}
