---
phase: 260711-lzb-make-hillshading-work-offline-in-the-flu
reviewed: 2026-07-11T00:00:00Z
depth: quick
files_reviewed: 12
files_reviewed_list:
  - app/lib/components/base/trail_map.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/models/map_cell.dart
  - app/lib/models/trail.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/services/trail_download_service.dart
  - app/lib/util/offline_style_rewriter.dart
  - app/test/util/offline_style_rewriter_test.dart
  - db/main.go
  - db/migrations/1781900000_added_dem_fields_to_tile_cells.go
  - db/routes/map_cells_id.go
  - db/services/tiles/generator.go
findings:
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 260711-lzb: Code Review Report

**Reviewed:** 2026-07-11T00:00:00Z
**Depth:** quick
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Reviewed the DEM/hillshading pipeline end to end: Go tile generator + routes + migration, and the Flutter model/entity/download-service/style-rewriter chain. The Dart-side `offline_style_rewriter.dart` rewrite (the bug this phase set out to fix — raster-dem sources no longer get repointed at vector `.pmtiles` archives) is solid and well covered by `offline_style_rewriter_test.dart`. The download service's cache-aware, best-effort DEM fetch is also mostly sound.

The one critical finding is on the Go side: `generator.go`'s `EnsureCell` was correctly extended with retry logic for a DEM archive that failed or is stuck (`status == "ready" && dem_status != "ready"` still falls through to `generateCell`, which re-attempts `generateDemCell` while skipping the already-extracted vector archive). But `MapCellsGet` — the only HTTP entry point that ever calls `EnsureCell` — short-circuits and returns the cached "ready" vector response as soon as `status == "ready"`, without ever invoking `EnsureCell` again. The retry path added in `generator.go` is therefore unreachable from any client-visible flow: a transient DEM extraction failure (or a server restart mid-extraction leaving `dem_status = "pending"`) permanently strands that cell without hillshade, and pre-existing (pre-migration) vector-ready cells are never backfilled with a DEM archive at all, even though the generator explicitly documents that backfill as a design goal.

## Critical Issues

### CR-01: DEM retry/backfill logic in `EnsureCell` is unreachable — a failed or never-attempted DEM extraction is permanently stuck

**File:** `db/routes/map_cells_id.go:23-36`
**Issue:**
`MapCellsGet` returns the cached "ready" response and returns early whenever `records[0].GetString("status") == "ready"` and the vector file exists on disk — regardless of `dem_status`:

```go
if len(records) > 0 && records[0].GetString("status") == "ready" {
    if _, err := os.Stat(tiles.CellPath(cell)); err == nil {
        resp := map[string]any{ "status": "ready", "download_url": "..." }
        if records[0].GetString("dem_status") == "ready" {
            if _, err := os.Stat(tiles.DemCellPath(cell)); err == nil {
                resp["dem_download_url"] = "..."
            }
        }
        return e.JSON(http.StatusOK, resp)   // <-- returns here, EnsureCell never called
    }
}
```

`tiles.EnsureCell` (`db/services/tiles/generator.go:49-83`) is *only* invoked from the fallback branch further down (line 45-49), which is unreachable once `status == "ready"`. Its regeneration logic:

```go
if record.GetString("status") == "ready" && record.GetString("dem_status") == "ready" {
    if _, err := os.Stat(CellPath(cell)); err == nil {
        if _, err := os.Stat(DemCellPath(cell)); err == nil {
            return nil
        }
    }
    log.Printf("[tiles] cell %s marked ready but file(s) missing, regenerating", cell.CacheKey())
}
return generateCell(app, record, cell)   // re-attempts DEM even when vector is already ready
```

...was clearly written with the intent that a vector-ready/DEM-not-ready cell gets retried on the next call (and `generateCell` correctly skips re-extracting the vector archive when `outputPath` already exists, only re-running `generateDemCell`). But that code path is dead from the client's perspective: once a cell's vector `status` first flips to `"ready"`, `MapCellsGet` never calls `EnsureCell` for that cell again for the lifetime of the deployment.

Consequences:
1. Any transient DEM extraction failure (`dem_status = "error"`, e.g. a `download.mapterhorn.com` hiccup, disk-full, timeout) permanently strands that grid cell without hillshade — no request ever retries it.
2. A server restart/crash mid-`generateDemCell` leaves `dem_status = "pending"` forever with the same effect.
3. Every tile cell that existed **before** this migration (vector-ready, no `dem_status` at all) will *never* get a DEM archive generated, despite `generateCell`'s own comment explicitly describing this as the "DEM-only refresh ... backfilling a DEM archive" scenario it was built to support (`db/services/tiles/generator.go:151-154`).

`MapCellsStatus` (`db/routes/map_cells_id.go:57-89`) has the identical gap: it only reads DB/FS state and never calls `EnsureCell`, so polling a "ready" cell can never trigger a DEM retry either.

**Fix:** Call `EnsureCell` (in the background, same fire-and-forget pattern already used lower in the function) whenever the vector is ready but `dem_status` is not `"ready"`, e.g.:

```go
if len(records) > 0 && records[0].GetString("status") == "ready" {
    if _, err := os.Stat(tiles.CellPath(cell)); err == nil {
        resp := map[string]any{
            "status":       "ready",
            "download_url": "/map/cells/" + cell.CacheKey() + "/download",
        }
        demReady := records[0].GetString("dem_status") == "ready"
        if demReady {
            if _, err := os.Stat(tiles.DemCellPath(cell)); err == nil {
                resp["dem_download_url"] = "/map/cells/" + cell.CacheKey() + "/download-dem"
            } else {
                demReady = false
            }
        }
        if !demReady {
            // Kick off (idempotent, inFlight-deduped) DEM backfill/retry —
            // never blocks this response; vector remains servable regardless.
            go func() { _ = tiles.EnsureCell(e.App, cell) }()
        }
        return e.JSON(http.StatusOK, resp)
    }
}
```

## Warnings

### WR-01: DEM download failure in `_downloadMapTiles` only guards against `DioException`, not all exceptions — can violate the documented "never fails the trail download" contract

**File:** `app/lib/services/trail_download_service.dart:205-221`
**Issue:** The function's own doc comment states the DEM archive is "best-effort ... a DEM failure or a missing `demDownloadUrl` must never fail the trail download." The inner guard only catches `DioException`:

```dart
String? demPath = demCached ? demLocalPath : null;
if (!demCached && readyCell?.demDownloadUrl != null) {
  try {
    await _api.download(readyCell!.demDownloadUrl!, demLocalPath, cancelToken: cancelToken);
    demPath = demLocalPath;
  } on DioException catch (e) {
    if (CancelToken.isCancel(e)) rethrow;
    if (await File(demLocalPath).exists()) {
      await File(demLocalPath).delete();
    }
    demPath = null;
  }
}
```

Any non-`DioException` thrown here (e.g. a `FileSystemException` while writing `demLocalPath`, or thrown by the cleanup `File(...).delete()` call itself) is not caught by this `on DioException` clause, escapes to the outer `try { ... } on DioException { ...; rethrow; }` block (which also only matches `DioException`), and propagates all the way out to `downloadTrail`'s generic `catch (e) { await trailDir.delete(recursive: true); rethrow; }` — deleting the *entire* trail download (photos, waypoints, vector tiles, everything) for what was supposed to be a purely cosmetic hillshade failure.

**Fix:** Broaden the inner catch to `catch (e)` (still re-throwing on cancellation) so any DEM-download failure degrades gracefully instead of failing the whole trail:

```dart
} catch (e) {
  if (e is DioException && CancelToken.isCancel(e)) rethrow;
  if (await File(demLocalPath).exists()) {
    await File(demLocalPath).delete();
  }
  demPath = null;
}
```

### WR-02: New `pmtiles extract` DEM subprocess has no timeout — a hung extraction blocks the per-cell mutex indefinitely

**File:** `db/services/tiles/generator.go:210-226`
**Issue:** `generateDemCell` runs `exec.Command("pmtiles", "extract", mapterhornSource, ...)` with no timeout/context, mirroring the pre-existing vector `exec.Command` but adding a second unbounded external-network-dependent subprocess call per cell. If `download.mapterhorn.com` hangs (rather than erroring), `demCmd.Run()` blocks forever, and — because `EnsureCell` holds the per-cell `sync.WaitGroup` in `inFlight` for the whole call — every other concurrent request for the same grid cell also blocks forever waiting on `wg.Wait()`.
**Fix:** Use `exec.CommandContext` with a bounded timeout (e.g. `context.WithTimeout(context.Background(), 5*time.Minute)`) for both the vector and DEM extraction commands so a hung remote source can't wedge a cell's `inFlight` entry permanently.

### WR-03: Silently swallowed `app.Save` errors on the DEM error path can leave `dem_status` stuck at `"pending"`

**File:** `db/services/tiles/generator.go:219-226`
**Issue:**
```go
if err := demCmd.Run(); err != nil {
    os.Remove(demOutputPath)
    log.Printf("[tiles] DEM extract failed for cell %s: %v", cell.CacheKey(), err)
    record.Set("dem_status", "error")
    record.Set("dem_error_message", err.Error())
    _ = app.Save(record)   // error discarded, not logged
    return
}
```
If `app.Save` itself fails here (DB write error), `dem_status` remains whatever it was set to at line 204-206 (`"pending"`), and — compounding CR-01 — there is no code path that will ever revisit this cell, since `MapCellsGet` only reacts to `dem_status == "ready"`/DB state and never re-triggers `EnsureCell` for a `"pending"`-forever cell either.
**Fix:** At minimum, log the `app.Save` error so operators can see it (`if err := app.Save(record); err != nil { log.Printf(...) }`), consistent with how the success path already checks and logs the `app.Save` error at line 240-243.

## Info

### IN-01: `dem_size_bytes` is persisted but never surfaced by any API response

**File:** `db/services/tiles/generator.go:238`, `db/routes/map_cells_id.go`
**Issue:** `generateDemCell` sets `record.Set("dem_size_bytes", fi.Size())`, but neither `MapCellsGet` nor `MapCellsStatus` ever reads/returns it (they only surface `dem_download_url`), and the Dart `MapCellStatusResponse` model (`app/lib/models/map_cell.dart`) has no corresponding field. This mirrors the vector `size_bytes` field, which *is* surfaced (`resp["size_bytes"] = ...` in `MapCellsStatus`), so the asymmetry looks like an oversight rather than intentional.
**Fix:** Either surface `dem_size_bytes` alongside `dem_download_url` in both route handlers (for parity with the vector field and so the Flutter download-progress UI could eventually show DEM size), or drop the field if it's not needed, to avoid a write-only column.

### IN-02: `demCellPaths`/`cellPaths` index correspondence is implicit and undocumented at the call site

**File:** `app/lib/util/offline_style_rewriter.dart:102-165`, `app/lib/services/trail_download_service.dart:139-241`
**Issue:** `_rewriteSourceGroup` labels extra cells `-cell-<i>` independently for the vector group (`cellPaths`) and the DEM group (`demCellPaths`). Because each `pmtiles://` archive is self-describing (it serves whatever bbox it was extracted for), this doesn't cause a functional bug today — but `trail_download_service.dart` builds `demCellPaths` via `results.map((r) => r.$2).whereType<String>().toList()`, which *compacts out* any cell with a missing DEM archive. If a future change assumes `cellPaths[i]` and `demCellPaths[i]` are the same geographic cell (e.g. to pair them up for some UI/debug purpose), that assumption would silently be wrong whenever a DEM download fails for one of several cells. Worth a comment near the `whereType<String>()` compaction noting that the two lists are intentionally independent and not index-aligned.
**Fix:** Add a short comment at `trail_download_service.dart:238-240` clarifying that `demCellPaths` is compacted (order not guaranteed to align with `cellPaths` by index) and that `offline_style_rewriter.dart` relies only on each archive's own embedded bounds, never on positional pairing.

---

_Reviewed: 2026-07-11T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
