package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionsList returns the merged config-plus-build-state catalog for every
// region configured in the admin's region catalog file (BACK-04). The admin
// config (regions.LoadRegionCatalog) is the source of truth for which
// regions exist — a region removed from config disappears from the response
// even if a stale region_archives record lingers.
//
// Auth is enforced at the route-group level in main.go (apis.RequireAuth()),
// so — unlike map_cells_id.go's unauthenticated /map/cells routes — every
// handler in this file (listing AND both download routes) requires a logged
// in user (D-07). An auth-gated listing with unauthenticated file bytes
// would defeat that requirement, so the download handlers below enforce the
// same posture at the group level too.
func RegionsList(e *core.RequestEvent) error {
	catalog, err := regions.LoadRegionCatalog()
	if err != nil {
		return e.InternalServerError("failed to load region catalog", err)
	}

	entries := make([]map[string]any, 0, len(catalog))
	for _, r := range catalog {
		records, _ := e.App.FindAllRecords("region_archives",
			dbx.NewExp("region_id = {:id}", dbx.Params{"id": r.ID}),
		)

		entry := map[string]any{
			"id":   r.ID,
			"name": r.Name,
			"bbox": []float64{r.Bbox[0], r.Bbox[1], r.Bbox[2], r.Bbox[3]},
		}

		if len(records) == 0 {
			// No build has ever started for this region yet — first build
			// pending, per D-08.
			entry["status"] = "building"
			entries = append(entries, entry)
			continue
		}

		record := records[0]
		status := record.GetString("status")
		entry["status"] = status

		if version := record.GetString("vector_built_date"); version != "" {
			entry["version"] = version
		}

		if status == "ready" {
			if _, err := os.Stat(regions.RegionArchivePath(r.ID)); err == nil {
				entry["vector_url"] = "/api/v1/regions/" + r.ID + "/download"
				entry["vector_size"] = int64(record.GetFloat("vector_size_bytes"))
			}
		}

		if status == "error" {
			entry["error"] = record.GetString("error_message")
		}

		demStatus := record.GetString("dem_status")
		if demStatus != "" {
			entry["dem_status"] = demStatus
		}
		if demStatus == "ready" {
			if _, err := os.Stat(regions.RegionDemPath(r.ID)); err == nil {
				entry["dem_url"] = "/api/v1/regions/" + r.ID + "/download-dem"
				entry["dem_size"] = int64(record.GetFloat("dem_size_bytes"))
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
