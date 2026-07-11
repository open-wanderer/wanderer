---
phase: quick-260711-lzb
verified: 2026-07-11T14:41:54Z
status: human_needed
score: 5/5 must-haves verified
overrides_applied: 0
---

# Quick Task 260711-lzb: Make hillshading work offline in the Flutter app Verification Report

**Task Goal:** Make hillshading work offline in the Flutter app
**Verified:** 2026-07-11T14:41:54Z
**Status:** human_needed
**Re-verification:** No — initial verification (post code-review fix commit `33c1c114`)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Downloading a trail also downloads a companion per-cell DEM `.pmtiles` archive from the server | ✓ VERIFIED | `db/services/tiles/generator.go:211-261` (`generateDemCell`) runs a second `pmtiles extract` against `mapterhornSource` per cell; `db/routes/map_cells_id.go:127-138` serves it via `GET /map/cells/{cellKey}/download-dem`; `app/lib/services/trail_download_service.dart:205-221` downloads it best-effort into `entity.demPmTiles` |
| 2 | The offline style points `hillshadeSource` at the local DEM archive (`pmtiles://file://…_dem.pmtiles`), not the vector archive | ✓ VERIFIED | `app/lib/util/offline_style_rewriter.dart:102-165` splits `raster-dem`-typed sources out of the vector `tiledKeys` path into `demKeys`, routed via `_pointDemSourceAtCell` to `demCellPaths`; confirmed against real style asset `app/assets/map/wanderer_light.json:12-15` which defines `hillshadeSource` as `type: raster-dem` |
| 3 | The offline DEM source carries `encoding:terrarium` and `tileSize:512` so relief decodes correctly | ✓ VERIFIED | `_pointDemSourceAtCell` (`offline_style_rewriter.dart:258-264`) sets `source['encoding']='terrarium'`, `source['tileSize']=512`, `source['maxzoom']=_offlineDemMaxZoom` (12); covered by test `raster-dem (hillshade) single cell` — passing |
| 4 | Opening a downloaded trail in airplane mode shows hillshade relief | ? UNCERTAIN (needs device test) | Code path is wired end-to-end and unit-verified, but this is a device-observable runtime behavior the plan itself flagged as `<human-check>` (A2/A3 smoke test). SUMMARY.md's own "Manual Verification Required" section confirms this has not yet been executed. |
| 5 | The vector basemap still renders offline with no regression (existing rewriter tests still pass) | ✓ VERIFIED | `flutter test test/util/offline_style_rewriter_test.dart` → 17/17 passing, including all 11 pre-existing vector-only tests unchanged |

**Score:** 5/5 code-verifiable truths pass; 1 requires human device testing (see Human Verification below).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/migrations/1781900000_added_dem_fields_to_tile_cells.go` | `dem_status`/`dem_size_bytes`/`dem_error_message` fields on `tile_cells` | ✓ VERIFIED | Additive migration confirmed; adds all three fields with `required:false`; Down func removes them |
| `db/services/tiles/generator.go` | Per-cell DEM pmtiles extraction against `download.mapterhorn.com` | ✓ VERIFIED | `DemCellPath`, `mapterhornSource`, `demMaxZoom=12`, `generateDemCell` all present; `go build`/`go vet` clean |
| `db/routes/map_cells_id.go` | `GET /map/cells/{cellKey}/download-dem` + `dem_download_url` in status JSON | ✓ VERIFIED | `MapCellsDownloadDem` present; `dem_download_url` conditionally added in both `MapCellsGet`/`MapCellsStatus` only when `dem_status=="ready"` and file exists on disk |
| `app/lib/util/offline_style_rewriter.dart` | raster-dem special-case pointing hillshade at DEM cells | ✓ VERIFIED | `_pointDemSourceAtCell`, demKeys routing, empty-DEM drop path all present and wired |
| `app/test/util/offline_style_rewriter_test.dart` | Tests proving raster-dem sources point at DEM archives with terrarium/512 | ✓ VERIFIED | 6 new tests present (single-cell, multi-cell, empty-DEM, 3x path-safety); all pass |
| `app/lib/services/trail_download_service.dart` | Parallel DEM cell download + `entity.demPmTiles` wiring | ✓ VERIFIED | `_downloadMapTiles` returns `(vector, dem)` record; `downloadTrail` assigns `entity.demPmTiles` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `trail_download_service.dart` | `/map/cells/<key>/download-dem` | `MapCellStatusResponse.demDownloadUrl` | ✓ WIRED | `readyCell.demDownloadUrl` deserializes `dem_download_url` (confirmed in `map_cell.freezed.dart`/`map_cell.g.dart`) and is used to download the DEM archive |
| `offline_style_rewriter.dart` | `pmtiles://file://<dem cell>` | raster-dem source pointing | ✓ WIRED | `_pointDemSourceAtCell` sets `url='pmtiles://file://$demPath'` |
| `generator.go demMaxZoom` | `offline_style_rewriter.dart _offlineDemMaxZoom` | shared max-zoom constant (must be equal) | ✓ VERIFIED | Both are `12`; each side's doc comment cross-references the other explicitly |
| `trail_map.dart` `_composeStyle` | `rewriteStyleForOffline(demCellPaths:)` | `widget.trail.demPmTiles` | ✓ WIRED | `trail_map.dart:154` |
| `navigation_screen.dart` `_composeStyle` | `rewriteStyleForOffline(demCellPaths:)` | `trailProvider(widget.id).value?.demPmTiles ?? []` | ✓ WIRED | `navigation_screen.dart:433-434`; confirmed via `git show d95b2c97` this is a clean targeted commit that does not touch the unrelated uncommitted `headingUp` puck hunks still sitting in the working tree |

### Code Review Follow-up Verification (CR-01, WR-01, WR-02, WR-03)

The REVIEW.md (2026-07-11) flagged one critical and three warning issues. Follow-up commit `33c1c114` ("fix(quick-260711-lzb): make DEM retry/backfill reachable, harden extraction") claims to fix all four. Verified directly against current file contents (not just the diff):

| ID | Issue | Status | Evidence |
|----|-------|--------|----------|
| CR-01 | DEM retry/backfill in `EnsureCell` unreachable from `MapCellsGet`/`MapCellsStatus` | ✓ FIXED | `db/routes/map_cells_id.go:29-45` (`MapCellsGet`) and `:90-104` (`MapCellsStatus`) now compute `demReady` and fire `go func(){ tiles.EnsureCell(...) }()` whenever the vector is ready but DEM isn't — confirmed present in current file, not just the commit diff |
| WR-01 | DEM download catch only handled `DioException`, non-Dio errors could wipe the whole trail download | ✓ FIXED | `trail_download_service.dart:214-220` now uses bare `catch (e)` with `if (e is DioException && CancelToken.isCancel(e)) rethrow` |
| WR-02 | `pmtiles extract` subprocess had no timeout, could wedge a cell's `inFlight` mutex forever | ✓ FIXED | `generator.go:38` adds `extractTimeout = 5 * time.Minute`; both vector (`:171-174`) and DEM (`:220-223`) `exec.Command` calls converted to `exec.CommandContext` with this timeout |
| WR-03 | `app.Save` error on the DEM error path silently discarded | ✓ FIXED | `generator.go:237-239` now logs the save error instead of `_ = app.Save(record)` |

`go build ./...` and `go vet ./...` both pass clean against the current tree (post-fix). All 7 SUMMARY-referenced commits (`3f67cf37`, `68501626`, `ab2be809`, `3d6ff5e1`, `d95b2c97`, `21a516a4`) plus the follow-up fix (`33c1c114`) confirmed present in `git log`.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go build/vet | `cd db && go build ./... && go vet ./...` | Clean, no output | ✓ PASS |
| Flutter rewriter unit tests | `cd app && flutter test test/util/offline_style_rewriter_test.dart` | 17/17 passing | ✓ PASS |
| Flutter analyze on touched files | `cd app && flutter analyze <7 files>` | "No issues found!" | ✓ PASS |
| build_runner regeneration is stable (no drift) | `cd app && dart run build_runner build --delete-conflicting-outputs` then `git status` | Built clean; `git status` shows zero new diffs beyond pre-existing unrelated files | ✓ PASS |
| `main.go` registers the download-dem route | `grep -n "download-dem" db/main.go` | `220: g.GET("/{cellKey}/download-dem", routes.MapCellsDownloadDem)` | ✓ PASS |
| `objectbox-model.json` persists `demPmTiles` | `grep -n "demPmTiles" app/lib/objectbox-model.json` | Field present | ✓ PASS |
| Lockstep constant check | `grep demMaxZoom generator.go` / `grep _offlineDemMaxZoom offline_style_rewriter.dart` | Both `= 12` | ✓ PASS |

### Anti-Patterns Found

None. Scanned all 12 phase-modified files for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` — zero matches.

## Human Verification Required

### 1. Device smoke test — hillshade relief renders offline (airplane mode)

**Test:** Download a trail with the app online (or via the existing device with connectivity), enable airplane mode, then open the downloaded trail's map/navigation screen.
**Expected:** Terrain shading (hillshade relief) is visible over the basemap; the basemap is not blank or showing garbage tiles.
**Why human:** This is a native rendering/runtime behavior (MapLibre's `raster-dem` decode of a locally-read `pmtiles://` webp-terrarium archive) that cannot be verified by static analysis or unit tests. The plan's own Task 3 `<verify><human-check>` explicitly calls this out, and the RESEARCH.md flags two `[ASSUMED]` items (MapLibre's default raster-dem encoding requiring the `terrarium` injection, and native webp-terrarium decode from a local pmtiles archive) that only a device test confirms. SUMMARY.md itself lists this as outstanding under "Manual Verification Required" — it has not yet been executed as of this verification.

## Gaps Summary

No code-level gaps. All Go and Flutter artifacts exist, are substantive (not stubs), and are wired end-to-end: server-side DEM extraction → download route → Flutter model/entity/download-service threading → offline style rewriter raster-dem special case → both callers (`trail_map.dart`, `navigation_screen.dart`). The prior code-review's one critical (CR-01) and three warning findings (WR-01/02/03) were all independently re-verified as fixed in commit `33c1c114`, not just claimed — the fixes are present in the current file contents, and `go build`/`go vet`/`flutter test`/`flutter analyze` all pass clean.

The only open item is the device-level smoke test (airplane-mode hillshade rendering), which the plan itself scoped as a human-check that cannot be satisfied by an autonomous agent. This is not a code defect — it is the final runtime confirmation step the plan always intended to require before calling the feature field-verified.

---

*Verified: 2026-07-11T14:41:54Z*
*Verifier: Claude (gsd-verifier)*
