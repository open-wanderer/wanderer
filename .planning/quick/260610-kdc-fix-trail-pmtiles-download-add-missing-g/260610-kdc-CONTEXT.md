# Quick Task 260610-kdc: Fix Trail PMTiles Download - Context

**Gathered:** 2026-06-10
**Status:** Ready for planning

<domain>
## Task Boundary

Fix four issues in the trail PMTiles map download system:
1. Add missing `generating` status to `MapCellStatus` enum (bug fix)
2. Download map cells in parallel instead of sequentially (performance)
3. Validate `cellKey` path param and `bbox` query param before forwarding to PocketBase (security)
4. Add download progress reporting and cancellation support via Dio `CancelToken` (UX)

Key files:
- `app/lib/services/trail_download_service.dart` — orchestrates download flow
- `app/lib/models/map_cell.dart` — MapCellStatus enum and freezed models
- `web/src/routes/api/v1/map/cells/+server.ts` — bbox list endpoint
- `web/src/routes/api/v1/map/cells/[cellKey]/+server.ts` — cell request endpoint
- `web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts` — status poll endpoint
- `web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts` — binary download endpoint

</domain>

<decisions>
## Implementation Decisions

### Generating enum case
- Add `@JsonValue('generating') generating` to `MapCellStatus` enum
- Handle it in the `_pollUntilReady` switch the same as `pending`/`isNew` (continue polling)
- Run `dart run build_runner build` to regenerate the `.g.dart` and `.freezed.dart` files

### Parallel cell downloads
- Mirror the existing photo download pattern using `Future.wait()` 
- Replace the sequential `for` loop in `_downloadMapTiles` with parallel tasks
- Each cell task independently polls until ready then downloads

### Input validation
- `cellKey`: validate against regex `^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$` in all three `[cellKey]` routes, return 400 if invalid
- `bbox`: validate it contains exactly 4 comma-separated finite numbers in `cells/+server.ts`, return 400 if invalid

### Progress and cancellation
- Add a `cancelToken` parameter to `downloadTrail` and `_downloadMapTiles` (optional, nullable)
- Pass `cancelToken` to each `_api.download()` and `_api.get()` call
- Add an `onProgress` callback parameter to report overall progress (cells downloaded / total cells)
- Keep the callback optional so existing callers don't break

### Claude's Discretion
- Exact regex pattern for cellKey validation
- Whether to also pass CancelToken through to photo downloads
- How to handle partial downloads on cancellation (delete incomplete file)

</decisions>

<specifics>
## Specific Ideas

- The `_pollUntilReady` switch in `trail_download_service.dart:138` needs a `generating` case
- The `Future.wait()` for photos at `trail_download_service.dart:162` is the exact pattern to replicate for cells
- `MapCellInfo` already has a `url` field (the request URL) — no model changes needed beyond the enum
- The download route at `download/+server.ts` already handles binary streaming correctly — only needs the cellKey validation guard added at the top

</specifics>

<canonical_refs>
## Canonical References

- No external specs — requirements fully captured in decisions above

</canonical_refs>
