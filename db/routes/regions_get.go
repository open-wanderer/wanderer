package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionsList returns the full group+leaf region catalog from the seeded
// `regions` table (CATALOG-01/02/03, EXTRACT-03), joined with each leaf's
// build state from `region_archives`. The `regions` table (not an
// admin-supplied config file) is the source of truth for which regions
// exist — the old region_config.json / REGION_CATALOG_CONFIG_PATH loader is
// fully retired (EXTRACT-02).
//
// Every row (both kind=group and kind=leaf) is returned, regardless of
// `enabled`, so a client can render group node names/labels for the whole
// tree (29-RESEARCH.md Assumption A1) — trimming is a trivial later filter,
// but omitting group rows now would mean re-adding data later. Each entry
// always carries the hierarchy fields id/name/kind/parent/path/depth; leaf
// rows additionally carry bbox/enabled and the existing build-state shape
// (status/version/vector_url/vector_size/dem_status/dem_url/dem_size/error),
// joined to region_archives on region_id == the leaf's path (A2 — path is
// the provably-unique seeded key, not the record's own opaque id).
//
// Auth is enforced at the route-group level in main.go (apis.RequireAuth()),
// so — unlike map_cells_id.go's unauthenticated /map/cells routes — every
// handler in this file (listing AND both download routes) requires a logged
// in user (D-07). An auth-gated listing with unauthenticated file bytes
// would defeat that requirement, so the download handlers below enforce the
// same posture at the group level too.
func RegionsList(e *core.RequestEvent) error {
	records, err := e.App.FindAllRecords("regions")
	if err != nil {
		return e.InternalServerError("failed to load region catalog", err)
	}

	entries := make([]map[string]any, 0, len(records))
	for _, r := range records {
		entry := map[string]any{
			"id":         r.Id,
			"name":       r.GetString("name"),
			"kind":       r.GetString("kind"),
			"parent":     r.GetString("parent"), // relation field's raw value = parent record id, "" for roots (A3)
			"path":       r.GetString("path"),
			"depth":      r.GetInt("depth"),
			"sort_order": r.GetInt("sort_order"), // D-07: sibling ordering hint, on every row (group and leaf)
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

		var bbox []float64
		_ = r.UnmarshalJSONField("bbox", &bbox)
		entry["bbox"] = bbox
		entry["enabled"] = r.GetBool("enabled")

		archiveRecords, _ := e.App.FindAllRecords("region_archives",
			dbx.NewExp("region_id = {:id}", dbx.Params{"id": path}),
		)

		if len(archiveRecords) == 0 {
			// No build has ever started for this region yet — first build
			// pending, per D-08.
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
// requests — Phase 23's resumable download engine (TILE-02) relies on this.
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
