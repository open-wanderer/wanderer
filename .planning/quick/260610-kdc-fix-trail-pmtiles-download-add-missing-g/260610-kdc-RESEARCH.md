# Quick Task 260610-kdc: Fix Trail PMTiles Download - Research

**Researched:** 2026-06-10
**Domain:** Flutter/Dart (Dio downloads, freezed codegen) + SvelteKit input validation
**Confidence:** HIGH (verified against existing codebase patterns + pinned package versions)

## Summary

All four fixes are small, localized changes to two files in `app/` and four `+server.ts` files in `web/`. The codebase already contains the exact patterns to mirror: parallel downloads via `Future.wait()` (in `_downloadPhotos`), inline guard-clause param validation (in `geocoding/search/+server.ts` and the existing `bbox` null-check in `cells/+server.ts`), and freezed enums with `@JsonValue` (already in `map_cell.dart`). No new packages are needed — Dio 5.9.2 ships `CancelToken` and `DioException` natively; `share_plus`/`qr_flutter` from CLAUDE.md are unrelated to this task.

**Primary recommendation:** Mirror the existing `_downloadPhotos` `Future.wait()` pattern for parallel cells; add a single nullable `CancelToken` threaded through every `_api.get`/`_api.download`; validate params with inline `if (...) return json({...},{status:400})` guards matching `geocoding/search`. Regenerate freezed/json with `dart run build_runner build --delete-conflicting-outputs`.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Generating enum:** Add `@JsonValue('generating') generating` to `MapCellStatus`; handle in `_pollUntilReady` switch identically to `pending`/`isNew` (continue polling); run `dart run build_runner build`.
- **Parallel downloads:** Mirror photo `Future.wait()` pattern; replace the sequential `for` loop in `_downloadMapTiles`; each cell task independently polls then downloads.
- **Input validation:** `cellKey` regex `^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$` in all three `[cellKey]` routes → 400 if invalid; `bbox` must be exactly 4 comma-separated finite numbers in `cells/+server.ts` → 400 if invalid.
- **Progress/cancellation:** Add nullable `cancelToken` param to `downloadTrail` and `_downloadMapTiles`; pass to each `_api.download()`/`_api.get()`; add optional `onProgress` callback reporting cells-done/total; keep callbacks optional so existing callers don't break.

### Claude's Discretion
- Exact regex pattern for cellKey validation.
- Whether to also pass `CancelToken` through to photo downloads.
- How to handle partial downloads on cancellation (delete incomplete file).

### Deferred Ideas (OUT OF SCOPE)
- None specified.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-1 | Add `generating` enum case | freezed `@JsonValue` pattern below; build_runner step |
| FIX-2 | Parallel cell downloads | `Future.wait()` pattern (mirrors `_downloadPhotos`) |
| FIX-3 | Validate `cellKey` + `bbox` | inline guard-clause pattern from `geocoding/search` |
| FIX-4 | Progress + cancellation | Dio `CancelToken`, aggregated counter, `DioExceptionType.cancel` |

## Standard Stack

No new packages required. Verified against `app/pubspec.yaml`:

| Library | Version | Purpose | Notes |
|---------|---------|---------|-------|
| dio | ^5.9.2 [VERIFIED: pubspec.yaml] | HTTP client, ships `CancelToken` + `DioException` | already the project HTTP client |
| freezed | ^3.2.5 [VERIFIED: pubspec.yaml] | immutable models + enum json | freezed v3 |
| freezed_annotation | ^3.1.0 [VERIFIED: pubspec.yaml] | annotations | — |
| build_runner | ^2.13.1 [VERIFIED: pubspec.yaml] | codegen runner | — |
| json_serializable / json_annotation | (transitive) | `@JsonValue` enum serialization | drives the `.g.dart` |

## Architecture Patterns

### Pattern 1: Parallel downloads with per-task error isolation (mirror `_downloadPhotos`)
**What:** Map each cell to an async task that try/catches internally and returns `String?` (path or null on failure). `Future.wait` runs them concurrently; filter nulls after.
**When:** Replacing the sequential `for` loop in `_downloadMapTiles`.
**Key insight:** Returning `null` on per-cell failure (instead of rethrowing) keeps one failed cell from aborting the whole `Future.wait`. This already-present pattern is exactly right — do NOT use `Future.wait(..., eagerError: true)` (that cancels siblings on first error).

```dart
// Source: existing app/lib/services/trail_download_service.dart:142-163 (photo pattern)
final downloadTasks = infoList.cells.map((cell) async {
  final key = cell.key;
  final localPath = '${tilesDir.path}/$key.pmtiles';
  if (await File(localPath).exists()) return localPath;
  try {
    final requestRes = await _api.get(cell.url, cancelToken: cancelToken);
    var ready = MapCellStatusResponse.fromJson(requestRes.data!);
    if (ready.status != MapCellStatus.ready) {
      ready = await _pollUntilReady(ready.statusUrl!, key, cancelToken);
    }
    await _api.download(ready.downloadUrl!, localPath, cancelToken: cancelToken);
    onProgress?.call(++completed, total); // see Pattern 3
    return localPath;
  } on DioException catch (e) {
    if (CancelToken.isCancel(e)) rethrow; // propagate cancellation, don't swallow
    print('Failed to download map cell $key: $e');
    return null;
  } catch (e) {
    print('Failed to download map cell $key: $e');
    return null;
  }
}).toList();
final results = await Future.wait(downloadTasks);
return results.whereType<String>().toList();
```
[ASSUMED — synthesized from existing codebase pattern; verify by running build]

### Pattern 2: Single CancelToken threaded through all calls
**What:** One nullable `CancelToken? cancelToken` parameter on `downloadTrail` / `_downloadMapTiles`, passed to every `_api.get()` and `_api.download()`. A single token cancels all in-flight requests at once when `token.cancel()` is called by the caller (e.g., a UI cancel button).
**When:** All Dio calls in the download flow.
```dart
Future<void> downloadTrail(Trail trail, {CancelToken? cancelToken,
    void Function(int done, int total)? onProgress}) async { ... }
await _api.get(url, cancelToken: cancelToken);
await _api.download(url, savePath, cancelToken: cancelToken);
```
[CITED: pub.dev/packages/dio — `cancelToken` is an optional named param on `get`/`download`]

### Pattern 3: Aggregated progress across parallel tasks
**What:** Report coarse-grained "cells completed / total cells" rather than per-byte progress (simpler, and stable across concurrent tasks). Increment a shared counter inside each task after its download finishes; invoke `onProgress?.call(completed, total)`.
**When:** Overall UX progress bar.
**Why counter not byte-sum:** With N concurrent downloads, byte-level `onReceiveProgress` callbacks interleave unpredictably and require summing per-cell received/total maps. A completed-cell counter is monotonic and race-safe in Dart's single-threaded event loop (no locks needed — increments run on the main isolate).
```dart
var completed = 0;
final total = infoList.cells.length;
// inside each task, after successful download:
onProgress?.call(++completed, total);
```
[ASSUMED — standard approach; Dart isolate single-threading makes `++completed` safe]

### Pattern 4: Inline guard-clause param validation (mirror `geocoding/search`)
**What:** Validate at the top of the handler, `return json({message}, {status:400})` early on failure. The codebase does NOT use Zod for path/query params — it uses plain guards. Match that.
**When:** All four `+server.ts` routes.
```typescript
// cellKey routes — add at top of GET, before pb.send
const cellKey = event.params.cellKey;
const CELL_KEY_RE = /^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$/;
if (!cellKey || !CELL_KEY_RE.test(cellKey)) {
  return json({ message: "Invalid cell key format" }, { status: 400 });
}

// cells/+server.ts — bbox: exactly 4 finite numbers
const parts = bbox.split(",");
if (parts.length !== 4 || parts.some((p) => !Number.isFinite(Number(p)))) {
  return json({ message: "Malformed bbox: expected 4 comma-separated numbers" }, { status: 400 });
}
```
[VERIFIED: web/src/routes/api/v1/geocoding/search/+server.ts uses this exact early-return guard style; bbox null-check already present in cells/+server.ts:64]

### Anti-Patterns to Avoid
- **`Future.wait(eagerError: true)`:** cancels sibling downloads on first failure — defeats per-cell isolation.
- **Swallowing cancellation as a generic failure:** if a `DioException` is a cancel, `rethrow` it (or stop the loop) — don't log it as "Failed to download" and continue, or cancellation won't actually stop the flow.
- **Zod schema for these params:** out of step with the codebase's inline-guard convention; adds a dependency import for no benefit here.
- **Regex anchoring miss:** without `^...$` anchors the cellKey regex would match substrings — keep both anchors.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cancellable HTTP | custom abort flags + checks | Dio `CancelToken` | native, cancels in-flight socket reads |
| Detecting cancellation | string-match the error message | `CancelToken.isCancel(e)` or `e.type == DioExceptionType.cancel` | robust, version-stable |
| Concurrency | manual `Completer` orchestration | `Future.wait` over a `.map()` | idiomatic, already used for photos |
| Enum JSON mapping | manual `switch` parse | freezed `@JsonValue` + build_runner | already generated |

## Common Pitfalls

### Pitfall 1: Non-exhaustive switch after adding `generating`
**What goes wrong:** Dart's exhaustive `switch` on `MapCellStatus` in `_pollUntilReady` (lines 118–128) will produce an **analyzer error** (not just a warning) once `generating` is added, because the enum gains a member with no matching case.
**How to avoid:** Add `case MapCellStatus.generating:` alongside `pending`/`isNew` (`continue`) in the **same** switch. This is actually a feature — the compiler forces you to handle it.
**Warning sign:** `dart analyze` reports "The type 'MapCellStatus' is not exhaustively matched."

### Pitfall 2: Stale generated files after enum edit
**What goes wrong:** Editing the enum without regenerating leaves `map_cell.g.dart` unable to deserialize `"generating"` → runtime parse failure.
**How to avoid:** Run `dart run build_runner build --delete-conflicting-outputs` from `app/`. The `--delete-conflicting-outputs` flag prevents the "conflicting outputs" prompt that stalls CI/non-interactive runs.
**Warning sign:** `Invalid argument` / `null` on `MapCellStatus` during JSON parse at runtime.

### Pitfall 3: Partial `.pmtiles` files left on cancellation
**What goes wrong:** Dio writes to the save path incrementally; cancelling mid-download leaves a truncated `.pmtiles` that later looks "downloaded" (the `File(localPath).exists()` guard would skip re-download and the map would render corrupt tiles).
**How to avoid (discretion item):** In the `on DioException` cancel branch, delete the partial file before rethrowing: `if (await File(localPath).exists()) await File(localPath).delete();`. Recommended — the existence-check resume logic makes orphaned partials dangerous.
**Warning sign:** corrupt/blank map tiles after a cancelled-then-retried download.

### Pitfall 4: Cancellation propagation through `Future.wait`
**What goes wrong:** If only some tasks rethrow the cancel and others return `null`, `Future.wait` still rejects with the first error — but ensure the caller awaits/handles it so `downloadTrail` surfaces the cancellation instead of writing a partial `TrailEntity`.
**How to avoid:** Let the cancel `DioException` propagate out of `_downloadMapTiles`; in `downloadTrail`, do not `box.put(entity)` if the tile download threw cancellation.

## Code Examples

### freezed enum: add `generating`
```dart
// app/lib/models/map_cell.dart
enum MapCellStatus {
  @JsonValue('new') isNew,
  @JsonValue('pending') pending,
  @JsonValue('generating') generating,   // NEW
  @JsonValue('ready') ready,
  @JsonValue('error') error,
}
```
Then: `cd app && dart run build_runner build --delete-conflicting-outputs`
[VERIFIED: pattern matches existing map_cell.dart:6-15]

### Detecting cancellation
```dart
} on DioException catch (e) {
  if (CancelToken.isCancel(e)) {
    // optional: delete partial file, then rethrow / break poll loop
    rethrow;
  }
  // genuine failure → log + null
}
```
[CITED: pub.dev/packages/dio — `CancelToken.isCancel(DioException)` and `DioExceptionType.cancel`]

### Poll loop respecting cancellation
```dart
Future<MapCellStatusResponse> _pollUntilReady(String statusUrl, String cellKey,
    [CancelToken? cancelToken]) async {
  final deadline = DateTime.now().add(_pollTimeout);
  while (DateTime.now().isBefore(deadline)) {
    await Future.delayed(_pollInterval);
    final res = await _api.get(statusUrl, cancelToken: cancelToken); // throws on cancel
    final data = MapCellStatusResponse.fromJson(res.data!);
    switch (data.status) {
      case MapCellStatus.ready: return data;
      case MapCellStatus.error:
        throw Exception('Map cell $cellKey failed: ${data.error ?? 'unknown'}');
      case MapCellStatus.pending:
      case MapCellStatus.isNew:
      case MapCellStatus.generating:   // NEW — continue polling
        continue;
    }
  }
  throw Exception('Timed out waiting for map cell $cellKey');
}
```
Note: passing `cancelToken` to the `_api.get` inside the loop makes an in-flight poll throw immediately on cancel; the `Future.delayed` between polls is not itself cancellable but resolves within `_pollInterval` (3s) worst case. Acceptable; no extra handling needed.
[ASSUMED — synthesized; verify with build + manual cancel test]

## Validation Architecture

No automated test framework is configured for the Flutter `app/` (no `test/` harness referenced for this service). Validation for this task is:
- **Build gate:** `cd app && dart run build_runner build --delete-conflicting-outputs` succeeds.
- **Analyzer gate:** `cd app && dart analyze` clean (will catch the non-exhaustive switch if `generating` case is missed).
- **Web gate:** `cd web && npm run check` (svelte-check) clean after `+server.ts` edits.
- **Manual smoke:** trigger a trail download, confirm parallel cells, confirm cancel stops it, confirm a malformed `cellKey`/`bbox` returns 400.

[VERIFIED: CLAUDE.md lists svelte-check via `svelte-check`; build_runner is the project codegen path]

## Security Domain

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V5 Input Validation | yes | regex guard on `cellKey`, numeric/shape guard on `bbox` — both early-return 400 |

**Threat addressed:** Unvalidated `cellKey`/`bbox` are interpolated into the PocketBase path/query (`pb.send(\`/map/cells/${cellKey}\`)`). Without validation a crafted `cellKey` could attempt path traversal or unexpected upstream routing. The regex restricts to the strict `lon_lat_lon_lat` numeric shape (no `/`, no `..`, no letters), neutralizing traversal and injection vectors. [VERIFIED: STRIDE-Tampering; controls match existing geocoding guard style]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `CancelToken.isCancel(e)` and `DioExceptionType.cancel` exist in Dio 5.9.2 | Code Examples | LOW — stable since Dio 4.x; verify at build |
| A2 | `++completed` is race-safe across `Future.wait` tasks | Pattern 3 | LOW — Dart single isolate, no true parallelism |
| A3 | Deleting partial file on cancel is the right resume behavior | Pitfall 3 | LOW — discretion item; matches existence-check guard |
| A4 | `--delete-conflicting-outputs` is the correct build_runner flag | Pitfall 2 | LOW — standard freezed workflow |

## Open Questions

1. **Should cancellation also tear down photo downloads?**
   - Discretion item. Recommendation: pass the same `cancelToken` to `_downloadPhotos` too, so one cancel stops everything. Low effort, consistent UX.

## Sources

### Primary (HIGH confidence)
- `app/lib/services/trail_download_service.dart` — existing photo `Future.wait` pattern, poll loop
- `app/lib/models/map_cell.dart` — existing freezed `@JsonValue` enum
- `web/src/routes/api/v1/geocoding/search/+server.ts` — inline guard-clause validation convention
- `web/src/routes/api/v1/map/cells/+server.ts` — existing bbox null-check
- `app/pubspec.yaml` — pinned versions (dio 5.9.2, freezed 3.2.5, build_runner 2.13.1)

### Secondary (MEDIUM confidence)
- pub.dev/packages/dio — CancelToken / DioException API (CITED; Context7 unavailable this session)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions read from pubspec; no new deps
- Architecture: HIGH — every pattern mirrors existing codebase code
- Pitfalls: HIGH — exhaustive-switch and stale-codegen are deterministic Dart behaviors
- Dio cancel API specifics: MEDIUM — training + pub.dev, Context7 not reachable

**Research date:** 2026-06-10
**Valid until:** 2026-07-10 (stable; pinned versions)
