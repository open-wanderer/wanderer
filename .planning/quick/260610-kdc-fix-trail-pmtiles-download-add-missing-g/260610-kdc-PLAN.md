---
phase: quick-260610-kdc
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/models/map_cell.dart
  - app/lib/models/map_cell.g.dart
  - app/lib/models/map_cell.freezed.dart
  - app/lib/services/trail_download_service.dart
  - web/src/routes/api/v1/map/cells/+server.ts
  - web/src/routes/api/v1/map/cells/[cellKey]/+server.ts
  - web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts
  - web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts
autonomous: true
requirements: [FIX-1, FIX-2, FIX-3, FIX-4]
must_haves:
  truths:
    - "Server responses with status generating deserialize into MapCellStatus.generating instead of crashing"
    - "Map cells for a trail download concurrently rather than one at a time"
    - "A caller can cancel an in-flight trail download and report cells-done/total progress"
    - "A malformed cellKey or bbox returns HTTP 400 before reaching PocketBase"
  artifacts:
    - path: "app/lib/models/map_cell.dart"
      provides: "MapCellStatus enum with generating case"
      contains: "generating"
    - path: "app/lib/services/trail_download_service.dart"
      provides: "Parallel cell download with CancelToken + onProgress"
      contains: "Future.wait"
    - path: "web/src/routes/api/v1/map/cells/[cellKey]/+server.ts"
      provides: "cellKey regex guard"
      contains: "status: 400"
  key_links:
    - from: "app/lib/services/trail_download_service.dart"
      to: "MapCellStatus.generating"
      via: "_pollUntilReady switch case"
      pattern: "MapCellStatus.generating"
    - from: "app/lib/services/trail_download_service.dart"
      to: "Dio _api.get / _api.download"
      via: "cancelToken named param"
      pattern: "cancelToken: cancelToken"
    - from: "web/src/routes/api/v1/map/cells/[cellKey]/+server.ts"
      to: "PocketBase /map/cells/:cellKey"
      via: "regex guard before pb.send"
      pattern: "CELL_KEY"
---

<objective>
Fix four issues in the trail PMTiles offline-map download flow:
1. Add the missing `generating` status to the `MapCellStatus` freezed enum so server "generating" responses parse correctly (FIX-1).
2. Download map cells in parallel instead of sequentially, mirroring the existing photo `Future.wait()` pattern (FIX-2).
3. Validate `cellKey` and `bbox` request params with inline guard clauses before forwarding to PocketBase (FIX-3).
4. Thread a nullable Dio `CancelToken` and an optional `onProgress` callback through the download flow so a UI can cancel and show progress (FIX-4).

Purpose: A `generating` server status currently crashes JSON parsing; sequential downloads are slow; unvalidated `cellKey`/`bbox` are interpolated into PocketBase paths (tampering/traversal risk); and there is no way to cancel or observe a long multi-cell download.

Output: Updated `map_cell.dart` (+ regenerated codegen), updated `trail_download_service.dart`, and validation guards in four `+server.ts` routes.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/quick/260610-kdc-fix-trail-pmtiles-download-add-missing-g/260610-kdc-CONTEXT.md
@.planning/quick/260610-kdc-fix-trail-pmtiles-download-add-missing-g/260610-kdc-RESEARCH.md

# Source files this plan modifies (read before editing)
@app/lib/models/map_cell.dart
@app/lib/services/trail_download_service.dart
@web/src/routes/api/v1/map/cells/+server.ts
@web/src/routes/api/v1/map/cells/[cellKey]/+server.ts
@web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts
@web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts

# Pattern references (read for convention, do not modify)
@web/src/routes/api/v1/geocoding/search/+server.ts
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add generating enum case and regenerate codegen (FIX-1)</name>
  <files>app/lib/models/map_cell.dart, app/lib/models/map_cell.g.dart, app/lib/models/map_cell.freezed.dart</files>
  <action>
Per D-01 (Generating enum case): Add a new member to the MapCellStatus enum in app/lib/models/map_cell.dart between pending and ready: annotate with @JsonValue('generating') and name it generating (matching the existing @JsonValue('new') isNew and @JsonValue('pending') pending style already in the file). Do NOT touch the freezed model classes (MapCellInfoList, MapCellInfo, MapCellStatusResponse) -- per RESEARCH.md only the enum changes.

After editing, regenerate the generated files by running build_runner from the app/ directory with --delete-conflicting-outputs (per RESEARCH.md Pitfall 2 -- the flag avoids the interactive conflict prompt that stalls non-interactive runs). This rewrites map_cell.g.dart so "generating" deserializes, and refreshes map_cell.freezed.dart.

Do not yet edit the _pollUntilReady switch -- that lives in Task 2. Note that until Task 2 lands, dart analyze will report the switch as non-exhaustive; that is expected and fixed in Task 2.
  </action>
  <verify>
    <automated>cd app && dart run build_runner build --delete-conflicting-outputs && grep -q "generating" lib/models/map_cell.g.dart</automated>
  </verify>
  <done>MapCellStatus has a generating member with @JsonValue('generating'); map_cell.g.dart references the generating mapping; build_runner completes without error.</done>
</task>

<task type="auto">
  <name>Task 2: Parallel downloads, cancellation, and progress in TrailDownloadService (FIX-2, FIX-4)</name>
  <files>app/lib/services/trail_download_service.dart</files>
  <action>
Per D-01: Add case MapCellStatus.generating: to the _pollUntilReady switch (alongside the existing case MapCellStatus.pending: case MapCellStatus.isNew: continue; block) so generating-status polls continue waiting, identically to pending/isNew. This also resolves the non-exhaustive-switch analyzer error introduced in Task 1.

Per D-04 (Progress/cancellation): Add two optional named parameters to downloadTrail: CancelToken? cancelToken and void Function(int done, int total)? onProgress. Keep them optional/nullable so the existing caller app/lib/components/trail/trail_dropdown.dart:85 (downloadTrail(trail)) keeps compiling unchanged. Add a matching CancelToken? cancelToken and the onProgress callback to the _downloadMapTiles signature, and add an optional [CancelToken? cancelToken] to _pollUntilReady. Pass cancelToken to every _api.get(...) and _api.download(...) call in the download flow (the _fetchCellList get, the per-cell request get, the poll get, and the binary download). Claude's discretion (per CONTEXT): also thread the same cancelToken into _downloadPhotos so a single cancel stops everything -- RESEARCH.md recommends this.

Per D-02 (Parallel downloads): Replace the sequential for (final cell in infoList.cells) loop in _downloadMapTiles with the Future.wait() pattern that mirrors _downloadPhotos (the existing photo task block). Map each cell to an async task returning String? (the local path, or null on per-cell failure). Inside each task: keep the existing File(localPath).exists() resume guard; do the request get, poll-if-not-ready, then download; on success increment a shared var completed = 0; counter and call onProgress?.call(++completed, total) where total = infoList.cells.length. Filter results with .whereType<String>().toList(). Do NOT use Future.wait(..., eagerError: true) (RESEARCH.md anti-pattern -- it cancels siblings on first failure).

Cancellation handling (RESEARCH.md Pitfall 3 + Pitfall 4): catch DioException specifically; if CancelToken.isCancel(e) is true, delete the partial localPath file if it exists (if (await File(localPath).exists()) await File(localPath).delete();) then rethrow so cancellation propagates out of Future.wait and _downloadMapTiles -- do not swallow it as a generic "Failed to download" and continue. For genuine (non-cancel) DioException and other catch (e), keep the existing log-and-return-null behavior so one failed cell does not abort the batch. In downloadTrail, ensure the tile-download await is not wrapped so as to swallow the rethrown cancellation before box.put(entity) -- a cancelled tile download must prevent writing a partial TrailEntity.

No new import is required: import 'package:dio/dio.dart'; is already present (used for Dio), and CancelToken/DioException come from the same package.
  </action>
  <verify>
    <automated>cd app && dart analyze lib/services/trail_download_service.dart lib/models/map_cell.dart</automated>
  </verify>
  <done>dart analyze is clean for the service and model (no non-exhaustive-switch error). _downloadMapTiles uses Future.wait over a .map(); every Dio call receives cancelToken: cancelToken; cancellation rethrows (with partial-file cleanup) while ordinary failures return null; onProgress is invoked with cells-done/total; the existing downloadTrail(trail) call site still compiles.</done>
</task>

<task type="auto">
  <name>Task 3: Validate cellKey and bbox params in map cell routes (FIX-3)</name>
  <files>web/src/routes/api/v1/map/cells/+server.ts, web/src/routes/api/v1/map/cells/[cellKey]/+server.ts, web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts, web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts</files>
  <action>
Per D-03 (Input validation), mirror the inline guard-clause convention from web/src/routes/api/v1/geocoding/search/+server.ts (early return json({ message }, { status: 400 })). Do NOT introduce Zod -- the codebase validates path/query params with plain guards (RESEARCH.md anti-pattern: Zod here is out of step with convention).

cellKey routes (all three: [cellKey]/+server.ts, [cellKey]/status/+server.ts, [cellKey]/download/+server.ts): immediately after reading const cellKey = event.params.cellKey; and before any pb.send/event.fetch call, add a guard using a module-level const CELL_KEY_RE set to the anchored regex from CONTEXT ^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$ (Claude's discretion allows refining the pattern, but keep both ^ and $ anchors per RESEARCH.md -- without anchors the regex matches substrings and defeats the traversal guard). If cellKey is falsy or fails CELL_KEY_RE.test(cellKey), return json({ message: "Invalid cell key format" }, { status: 400 }). The download/+server.ts route currently imports only RequestEvent and handleError -- add json to its @sveltejs/kit import so the 400 guard can use it.

bbox route (cells/+server.ts): the existing null-check stays. After it, add a shape guard: split bbox on comma; if it does not have exactly 4 parts, or any part is not a finite number (!Number.isFinite(Number(p))), return json({ message: "Malformed bbox: expected 4 comma-separated numbers" }, { status: 400 }).

This neutralizes path-traversal/injection through cellKey (interpolated into the PocketBase path) and malformed bbox forwarding -- see threat T-quick-01.
  </action>
  <verify>
    <automated>cd web && grep -q "status: 400" "src/routes/api/v1/map/cells/+server.ts" && grep -lq "CELL_KEY" "src/routes/api/v1/map/cells/[cellKey]/+server.ts" "src/routes/api/v1/map/cells/[cellKey]/status/+server.ts" "src/routes/api/v1/map/cells/[cellKey]/download/+server.ts" && npm run check</automated>
  </verify>
  <done>All three [cellKey] routes reject a non-matching cellKey with 400 before calling PocketBase; cells/+server.ts rejects a bbox that is not 4 finite numbers with 400; npm run check (svelte-check) passes with no new errors.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Flutter app -> SvelteKit API | Untrusted cellKey path param and bbox query param cross here |
| SvelteKit API -> PocketBase backend | cellKey is interpolated into the PocketBase request path |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-01 | Tampering | map/cells/[cellKey] routes (path interpolated into pb.send) | mitigate | Anchored regex guard CELL_KEY_RE on cellKey; 400 before pb.send/event.fetch -- blocks path traversal and injection (ASVS V5 input validation) |
| T-quick-02 | Tampering | map/cells/+server.ts (bbox forwarded to PocketBase query) | mitigate | Shape guard: exactly 4 comma-separated finite numbers; 400 otherwise |
| T-quick-03 | Denial of Service | parallel cell downloads | accept | Cell count is bounded by trail bounds; single CancelToken lets the caller abort; no unbounded fan-out introduced |
</threat_model>

<verification>
- App codegen + analyzer: `cd app && dart run build_runner build --delete-conflicting-outputs && dart analyze lib/services/trail_download_service.dart lib/models/map_cell.dart` is clean.
- Web type check: `cd web && npm run check` passes with no new errors.
- Manual smoke (per RESEARCH.md Validation Architecture; no Flutter test harness configured): trigger a trail download, confirm cells download concurrently, confirm a CancelToken cancel stops the flow and leaves no partial .pmtiles, confirm a malformed cellKey and a malformed bbox each return 400.
</verification>

<success_criteria>
- MapCellStatus.generating exists and a "generating" server response deserializes without error (FIX-1).
- _downloadMapTiles runs cell downloads via Future.wait, not a sequential loop (FIX-2).
- All four routes reject malformed cellKey/bbox with HTTP 400 before reaching PocketBase (FIX-3).
- downloadTrail accepts optional cancelToken + onProgress, threads cancelToken through all Dio calls, reports cells-done/total, and the existing downloadTrail(trail) caller still compiles (FIX-4).
- dart analyze and npm run check are both clean.
</success_criteria>

<output>
Create `.planning/quick/260610-kdc-fix-trail-pmtiles-download-add-missing-g/260610-kdc-SUMMARY.md` when done.
</output>
