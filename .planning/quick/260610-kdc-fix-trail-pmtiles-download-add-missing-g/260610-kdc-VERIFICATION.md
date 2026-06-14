---
phase: quick-260610-kdc
verified: 2026-06-10T00:00:00Z
status: human_needed
score: 4/4 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Trigger a trail download and observe parallel cell downloads"
    expected: "Multiple cells download concurrently (not sequentially), completion is faster than sequential"
    why_human: "Flutter app must be running; parallel behaviour is not verifiable by static analysis"
  - test: "Cancel an in-flight trail download via CancelToken"
    expected: "Download stops promptly, no partial .pmtiles files remain on disk"
    why_human: "Requires live Dio cancel flow in running app; file cleanup confirmed by grep but runtime path needs human confirmation"
  - test: "Send GET /api/v1/map/cells/[cellKey] with a malformed cellKey (e.g. ../../../etc/passwd)"
    expected: "HTTP 400 with body {\"message\":\"Invalid cell key format\"} is returned before any PocketBase call"
    why_human: "Route guard verified by code inspection; runtime HTTP response needs manual curl/Postman confirmation"
  - test: "Send GET /api/v1/map/cells with a malformed bbox (e.g. bbox=a,b,c,d or bbox=1,2,3)"
    expected: "HTTP 400 with body {\"message\":\"Malformed bbox: expected 4 comma-separated numbers\"}"
    why_human: "Guard logic verified by code inspection; runtime HTTP response needs manual confirmation"
  - test: "Check onProgress callback fires for each completed cell during a multi-cell download"
    expected: "Callback called N times with incrementing done values and constant total"
    why_human: "Requires live download with multiple cells; cannot simulate in static analysis"
---

# Quick Task: Fix Trail PMTiles Download — Verification Report

**Task Goal:** Fix trail PMTiles download: add missing generating enum case, parallel cell downloads, cellKey and bbox input validation, and download progress with cancellation
**Verified:** 2026-06-10
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Server responses with status `generating` deserialize into `MapCellStatus.generating` instead of crashing | VERIFIED | `app/lib/models/map_cell.dart` line 11–12: `@JsonValue('generating') generating` added between `pending` and `ready`. `app/lib/models/map_cell.g.dart` line 37: `MapCellStatus.generating: 'generating'` in `_$MapCellStatusEnumMap`. `$enumDecode` uses this map for both encode and decode. |
| 2 | Map cells for a trail download concurrently rather than one at a time | VERIFIED | `trail_download_service.dart` lines 80–124: cells are mapped to async tasks via `.map((cell) async {...}).toList()` then awaited via `Future.wait(downloadTasks)` at line 126. Not sequential. |
| 3 | A caller can cancel an in-flight trail download and report cells-done/total progress | VERIFIED | `downloadTrail` signature (lines 22–25) accepts `CancelToken? cancelToken` and `void Function(int done, int total)? onProgress`. `cancelToken` is threaded to `_downloadPhotos` (line 40), `_downloadMapTiles` (line 46), `_fetchCellList` (line 69), per-cell `_api.get` (line 92), `_pollUntilReady` (line 100), `_api.download` (line 107), and poll `_api.get` (line 152). Cancellation rethrows after partial-file cleanup (lines 112–116). `onProgress?.call(++completed, total)` fires on already-cached cell (line 85) and on successful download (line 109). |
| 4 | A malformed cellKey or bbox returns HTTP 400 before reaching PocketBase | VERIFIED | All three `[cellKey]` routes define `CELL_KEY_RE = /^-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+_-?\d+\.\d+$/` at module scope and guard at the top of `GET` before any `pb.send`/`event.fetch` call. `cells/+server.ts` guards bbox with null-check (line 65) then shape check (lines 69–71). Both return `{ status: 400 }`. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/models/map_cell.dart` | MapCellStatus enum with `generating` case | VERIFIED | `@JsonValue('generating') generating` present at line 11 |
| `app/lib/models/map_cell.g.dart` | Codegen references `generating` mapping | VERIFIED | `MapCellStatus.generating: 'generating'` at line 37 |
| `app/lib/services/trail_download_service.dart` | Parallel cell download with CancelToken + onProgress | VERIFIED | `Future.wait` at line 126; `cancelToken` on every Dio call; `onProgress` at lines 85, 109 |
| `web/src/routes/api/v1/map/cells/[cellKey]/+server.ts` | cellKey regex guard returning 400 | VERIFIED | `CELL_KEY_RE` at line 4; guard at lines 62–64 |
| `web/src/routes/api/v1/map/cells/[cellKey]/status/+server.ts` | cellKey regex guard returning 400 | VERIFIED | `CELL_KEY_RE` at line 4; guard at lines 55–57 |
| `web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts` | cellKey regex guard returning 400; `json` imported | VERIFIED | `CELL_KEY_RE` at line 4; guard at lines 46–48; `json` imported at line 2 |
| `web/src/routes/api/v1/map/cells/+server.ts` | bbox shape guard returning 400 | VERIFIED | Guard at lines 69–71 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `trail_download_service.dart` | `MapCellStatus.generating` | `_pollUntilReady` switch case | VERIFIED | Line 164: `case MapCellStatus.generating:` falls through to `continue` alongside `pending` and `isNew` |
| `trail_download_service.dart` | `Dio _api.get` / `_api.download` | `cancelToken: cancelToken` named param | VERIFIED | Lines 92, 107, 137, 152, 188 all pass `cancelToken: cancelToken` |
| `web/src/routes/api/v1/map/cells/[cellKey]/+server.ts` | PocketBase `/map/cells/:cellKey` | regex guard before `pb.send` | VERIFIED | `CELL_KEY_RE.test(cellKey)` guard at line 62 precedes `pb.send` at line 67 |

### Data-Flow Trace (Level 4)

Not applicable — this phase modifies a Flutter service and TypeScript API routes, not data-rendering components. No dynamic data render path to trace.

### Behavioral Spot-Checks

Step 7b: SKIPPED — Flutter app requires a running device/emulator; web routes require a running SvelteKit + PocketBase stack. Static checks completed above are the available automated evidence.

### Probe Execution

Step 7c: No probe scripts found for this quick task directory. Plan `<verification>` block specifies `dart analyze` and `npm run check`; these are live-toolchain commands not executable without a full build environment. Treated as human verification items.

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|-------------|-------------|--------|----------|
| FIX-1 | Add `generating` enum case to `MapCellStatus` | SATISFIED | `map_cell.dart` line 11; `map_cell.g.dart` line 37 |
| FIX-2 | Parallel cell downloads via `Future.wait` | SATISFIED | `trail_download_service.dart` line 126 |
| FIX-3 | Validate `cellKey` and `bbox` before PocketBase | SATISFIED | All four `+server.ts` routes have guards returning 400 |
| FIX-4 | `CancelToken` + `onProgress` through download flow | SATISFIED | `downloadTrail` signature lines 22–25; threading confirmed across 8 call sites |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `trail_download_service.dart` | 191 | `// print(...)` commented-out debug log | Info | Non-blocking; leftover from photo download, pre-existing, not introduced by this phase |

No `TBD`, `FIXME`, `XXX`, or `HACK` markers found in any file modified by this phase. No stub implementations (empty returns, placeholder bodies) found.

### Existing Call Site Compatibility

`app/lib/components/trail/trail_dropdown.dart` line 85 calls `trailDownloadService.downloadTrail(trail)` with no named arguments. The updated signature has `cancelToken` and `onProgress` as optional named parameters — this call remains valid and compiles unchanged.

### Human Verification Required

1. **Parallel download behaviour**

   **Test:** Trigger a trail download on a multi-cell trail and observe network activity
   **Expected:** Multiple .pmtiles cell requests are in-flight simultaneously, not serialised
   **Why human:** Concurrency is correct by code structure (`Future.wait` over mapped async tasks) but runtime scheduling requires a live app to observe

2. **Cancellation stops download and cleans partial files**

   **Test:** Cancel a download mid-flight using a `CancelToken`
   **Expected:** All pending Dio calls abort; any partially-written `.pmtiles` file is deleted from the tiles directory
   **Why human:** File cleanup confirmed by code at lines 113–115 but runtime execution path needs device confirmation

3. **cellKey 400 rejection (live HTTP)**

   **Test:** `curl -v "http://localhost:5173/api/v1/map/cells/../../etc/passwd"`
   **Expected:** HTTP 400, body `{"message":"Invalid cell key format"}`, PocketBase never contacted
   **Why human:** Code guard is present and correct by inspection; runtime response needs confirmation

4. **bbox 400 rejection (live HTTP)**

   **Test:** `curl -v "http://localhost:5173/api/v1/map/cells?bbox=a,b,c,d"` and `curl -v "http://localhost:5173/api/v1/map/cells?bbox=1,2,3"`
   **Expected:** HTTP 400, body `{"message":"Malformed bbox: expected 4 comma-separated numbers"}`
   **Why human:** Code guard confirmed; runtime response needs confirmation

5. **onProgress fires correctly**

   **Test:** Call `downloadTrail(trail, onProgress: (done, total) => print('$done/$total'))` for a trail with N cells
   **Expected:** Callback fires N times with `done` incrementing from 1 to N and `total` constant
   **Why human:** Counter logic confirmed statically at lines 77–85, 109; runtime sequencing under `Future.wait` needs live observation

### Gaps Summary

No gaps found. All four must-have truths are verified by code inspection at all three levels (exists, substantive, wired). The `human_needed` status reflects that runtime/behavioural confirmation (parallel scheduling, cancellation execution, live HTTP response codes) cannot be established through static analysis alone.

---

_Verified: 2026-06-10_
_Verifier: Claude (gsd-verifier)_
