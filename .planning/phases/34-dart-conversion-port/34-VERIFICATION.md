---
phase: 34-dart-conversion-port
verified: 2026-08-01T08:45:38Z
status: gaps_found
score: 4/6 must-haves verified
overrides_applied: 0
gaps:
  - truth: "SC1: The app derives a draft trail's name, description, waypoints, start coordinates, date, distance, elevation gain/loss, duration, and bounding box from a GPX with no network call"
    status: failed
    reason: "Two confirmed defects break this for real, non-trivial inputs. (1) Enabling either post-capture toggle (recalcHeights/followRoads) on a file import silently discards the imported GPX's waypoints, name, and description before the trail is built AND before the stripped document is re-uploaded — permanent data loss, not just a draft-preview glitch. (2) The sanitize pass wired into parseGpxSafely only guards <ele> and <time>; package:gpx's GpxReader still throws FormatException on <hdop>, <vdop>, <pdop>, <magvar>, <geoidheight>, <ageofdgpsdata>, <sat>, <dgpsid> when empty/non-numeric — common in real exporter output (Garmin, Locus, OsmAnd) — so those files fail to import at all, a regression against the pre-phase server path (xml2js never threw)."
    artifacts:
      - path: "app/lib/util/route_planner_handoff_util.dart"
        issue: "mergeHeightsIntoGpx (lines ~34-63) builds a bare Gpx() carrying only trks — no metadata, wpts, or rtes. Its caller in trail_import_util.dart:120-137 assigns the result directly to finalGpx with no merge-back of the source document's gpx.metadata/gpx.wpts/gpx.rtes, and finalGpxData is re-serialised from that stripped document before upload."
        issue2: "No test exercises the recalcHeights:true or followRoads:true import path with a fixture containing <wpt>/<metadata><name> — trail_import_util_test.dart:361-388 only covers the (false,false) branch."
      - path: "app/lib/util/gpx_conversion_util.dart"
        issue: "sanitizeGpxEleAndTime (lines 30-52) rewrites only <ele> and <time>. gpx_reader.dart's _readDouble/_readInt (package:gpx 2.3.0, verified at ~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/gpx_reader.dart:339-347) call double.parse/int.parse with no try/tryParse guard for hdop, vdop, pdop, magvar, geoidheight, ageofdgpsdata (_readDouble) and sat, dgpsid (_readInt) — 8 more unguarded numeric tags."
    missing:
      - "Carry gpx.metadata / gpx.wpts / gpx.rtes (and trk name/desc) from the original parsed document onto the transformed GPX before building the trail in the recalc/follow-roads import branch, and add a regression test with waypoints + metadata name through that branch."
      - "Generalise the sanitize pass to cover all numerically-parsed GPX tags (or replace with a generic try/catch fallback), and add corpus/unit coverage per newly-covered tag."
  - truth: "SC4: POST /api/v1/trail/convert transcodes kml/kmz/tcx/fit to GPX and returns it without computing a trail, and its published OpenAPI description matches the new behavior"
    status: failed
    reason: "The transcode-only behavior and OpenAPI update are real and verified, but plan 34-07 removed the endpoint's only content validation as a side effect. The pre-phase handler wrapped gpx2trail() in try/catch and 400'd on unparseable content (git show ca063023); the new handler has no equivalent check on the JSON/raw-text branches — any non-empty body is echoed verbatim as `application/gpx+xml` with no nosniff/Content-Disposition. Combined with the endpoint being unauthenticated (not in privateRoutes) and /api/v1 being CSRF-exempt (hooks.server.ts:196, pre-existing) — both pre-existing conditions — this phase's change newly makes the endpoint a verbatim reflector of unvalidated, attacker-controlled bytes under an XML content type, which it was not before (the pre-phase success response was always a computed Trail JSON, never raw input echoed back)."
    artifacts:
      - path: "web/src/routes/api/v1/trail/convert/+server.ts"
        issue: "Lines 63-79: the JSON and raw-text branches perform no format validation before returning gpxData verbatim; convert.test.ts:36-47 and 124-135 assert only the empty-body 400 and the verbatim-echo success path — no test asserts a non-GPX/malformed body is rejected."
    missing:
      - "Re-validate gpxData (e.g. attempt a GPX parse) before returning 200, restoring a 400 on unparseable content."
      - "Add X-Content-Type-Options: nosniff (and ideally Content-Disposition: attachment) to the success response."
      - "Add a test asserting a non-GPX raw body 400s."
deferred: []
human_verification: []
---

# Phase 34: Dart Conversion Port Verification Report

**Phase Goal:** The app computes a trail's name, waypoints, distance, elevation, duration, and bounding box from a GPX entirely on-device — for recordings, route-planner output, and file imports — proven identical to the corrected web implementation; the server's convert endpoint stops computing trails at all.
**Verified:** 2026-08-01T08:45:38Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria 1-6)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | App derives name, description, waypoints, start coords, date, distance, elevation, duration, bounding box from a GPX with no network call | ✗ FAILED | Core happy path verified (corpus tests, `trailFromGpx` code inspection), but two confirmed defects break it for real inputs: (a) import with recalcHeights/followRoads discards waypoints/name/description permanently (`route_planner_handoff_util.dart` `mergeHeightsIntoGpx` + `trail_import_util.dart:120-137`); (b) 8 unguarded numeric GPX tags still crash the import path (`gpx_reader.dart` `_readDouble`/`_readInt` vs `sanitizeGpxEleAndTime`'s ele/time-only coverage). See gaps. |
| 2 | Shared fixture test proves Dart/TS produce identical metrics, covering CONV-01..05 | ✓ VERIFIED | `npx vitest run src/lib/models/gpx/gpx-corpus.test.ts` → 30/30 pass; `flutter test test/util/gpx_corpus_test.dart` → 30/30 pass. 10 on-disk fixtures at `fixtures/gpx-corpus/`, tolerances `1e-6`m/`1e-9`deg match D-03 in both suites. Independently re-derived fixture 01's expected distance in a fresh Python haversine calc: `134.59240148587796` — bit-identical to `expected.json`, confirming the corpus is a real cross-check, not seeded-and-copied theater. Minor completeness gaps not blocking this truth: WR-05 (tautological waypoint-icon assertion in the Dart suite) and WR-06 (malformed non-empty `<time>` diverges TS/Dart and is uncovered by any fixture) — both real, neither invalidates the CONV-01..05 coverage this truth requires. |
| 3 | Recordings, route-planner output, and .gpx file imports produce their trail through the Dart path — `POST /trail/convert` called for none of them | ✓ VERIFIED | `grep -rn "trail/convert" app/lib/` → exactly 1 match, in `transcodeToGpx` (`trail_import_util.dart:175`), reached only for non-GPX extensions. Machine-checked by `trail_import_util_test.dart`'s PORT-03 gate test (confirmed present, non-vacuous per its own summary's described bogus-match check). `buildDraftTrail`/`_saveRecordedTrack`/`importTrailFile` all call local `trailFromGpx`/`buildLocalTrail`, no HTTP round trip for GPX. |
| 4 | `POST /api/v1/trail/convert` transcodes kml/kmz/tcx/fit to GPX without computing a trail; OpenAPI description matches | ✗ FAILED | Transcode-only behavior confirmed by direct code read of `+server.ts` (no `gpx2trail`/`Trail` import, response is raw `gpxData`) and by running `convert.test.ts` conceptually against the current handler. OpenAPI JSDoc updated to describe `application/gpx+xml`. **However**, this same rewrite (plan 34-07) deleted the endpoint's only content validation with no replacement — confirmed by diffing against the pre-phase handler (`git show ca063023:.../+server.ts`), which 400'd via a `gpx2trail` try/catch that no longer exists. The endpoint now echoes arbitrary non-empty bytes verbatim as `application/gpx+xml`. See gaps. |
| 5 | Importing kml/kmz/tcx/fit while online still produces a correct trail from the server-transcoded GPX | ✓ VERIFIED | `transcodeToGpx` posts to `/trail/convert`, returns raw GPX string (tolerates both raw-string and legacy-JSON shapes for skew), and `buildLocalTrail`→`trailFromGpx` measures it on-device. `convert.test.ts`'s KML→GPX test confirms server-side transcoding still functions. Same CR-02 unguarded-tag exposure as Truth 1 applies to server-transcoded GPX too, but is already counted under Truth 1's gap — not double-counted here. |
| 6 | A recorded trail reports moving time (elapsed minus pause); an imported file continues to report elapsed time | ✓ VERIFIED | `_saveRecordedTrack` passes `movingDuration: navStats.elapsed` (`navigation_screen.dart:861`); `buildLocalTrail`/import path passes no `movingDuration` (stays `null`). Display rule verified identical in both languages: `web/src/lib/util/format_util.ts:27-34` and `app/lib/util/format_util.dart:15-21`, both `moving_duration > 0 ? moving_duration : duration`, wired at all rendering call sites (`trail_card`, `trail_list_item`, `trail_panel`, `trail_info_panel`, `trail_table`). `updateTotals()` on the web edit page (`+page.svelte:1607-1616`) writes only `distance/duration/elevation_gain/elevation_loss` — structurally cannot touch `moving_duration` (D-10 confirmed). Minor edge-case gap not blocking this truth: WR-08 — a zero-length recording persists `moving_duration: 0` instead of `null` because `navStats.elapsed` is passed unconditionally and the write guard is `!= null`, not `> 0`. |

**Score:** 4/6 truths verified

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| PORT-01 | 34-01, 34-04 | On-device draft trail computation, no network call | ✗ BLOCKED | Blocked by the same CR-01/CR-02 defects as Truth 1 — the on-device computation itself (`trailFromGpx`/`computeTrailMetrics`) is correct, but the app-level pipeline around it loses data or crashes on real inputs before computation can even run cleanly on some imports. |
| PORT-02 | 34-01, 34-03, 34-04 | Shared fixture test proves Dart/TS parity, covering CONV-01..05 | ✓ SATISFIED | Both suites pass (30/30 TS, 30/30 Dart), 10 fixtures, tolerances documented, independently spot-checked. |
| PORT-03 | 34-05, 34-06 | All three capture paths use the Dart path; `/trail/convert` called for none | ✓ SATISFIED | Single remaining call site confirmed via grep + machine-checked test. |
| PORT-04 | 34-07 | Convert endpoint transcode-only; OpenAPI matches | ✗ BLOCKED | Transcode-only behavior itself is real, but delivered with a newly-introduced unauthenticated content-reflection defect (CR-04) that a complete/safe implementation of this requirement should not carry. |
| PORT-05 | 34-05, 34-07 | Non-GPX import while online still produces a correct trail | ✓ SATISFIED (with shared CR-02 exposure noted under PORT-01) | `convert.test.ts` KML transcoding test + `buildLocalTrail` local computation confirmed. |
| CONV-06 | 34-02, 34-04, 34-05 | Recorded trail reports moving time; imported file reports elapsed | ✓ SATISFIED | Confirmed end to end; WR-08 zero-duration edge case is a narrow gap, not a blocker of the core requirement. |

No orphaned requirements — all 6 IDs (PORT-01..05, CONV-06) declared across the 7 plans match `.planning/REQUIREMENTS.md`'s Phase 34 mapping exactly.

### Adjudication of the 4 Reviewer-Reported Critical Findings

Each was independently re-derived from source, not taken on the reviewer's word:

| ID | Reviewer claim | My verdict | Basis |
|---|---|---|---|
| CR-01 | `mergeHeightsIntoGpx` builds a bare `Gpx()`, dropping wpts/metadata/description on a corrected import, and the stripped doc is uploaded | **CONFIRMED — real, phase-introduced, BLOCKER** | Read `mergeHeightsIntoGpx` (`route_planner_handoff_util.dart:34-63`): returns `Gpx()` with only `trks` set. Read `trail_import_util.dart:120-145`: `finalGpx` is reassigned to this bare object with no merge of `gpx.metadata`/`gpx.wpts`/`gpx.rtes`, and `finalGpxData = GpxWriter().asString(finalGpx)` is what gets uploaded. Read `trailFromGpx` (`gpx_conversion_util.dart:421-471`): reads exactly those now-empty fields for name/description/waypoints. No test covers the toggled-on branch with a fixture carrying waypoints/metadata. |
| CR-02 | Sanitizer covers 4 of 12 unguarded `parse` sites; common tags still crash | **CONFIRMED — real, phase-introduced, BLOCKER** | Read `sanitizeGpxEleAndTime` (only `<ele>`/`<time>` regexes). Read the actual installed `gpx` 2.3.0 package source at `~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/gpx_reader.dart:339-347`: `_readDouble`/`_readInt` call `double.parse`/`int.parse` (throwing, not `tryParse`), reached for `hdop`, `vdop`, `pdop`, `magvar`, `geoidheight`, `ageofdgpsdata`, `sat`, `dgpsid` — 8 tags, matching the reviewer's count. |
| CR-03 | `parseGpxSafely` isn't applied at `trail_provider.dart:42-43` and `trail_entity.dart:187`, contradicting its own doc comment | **CONFIRMED — real, but NOT a phase-34 must-have; scoped as WARNING, not BLOCKER** | `grep` confirms both sites still call `GpxReader().fromString(sanitizeGpxEmail(...))` directly. These are pre-existing call sites (server-downloaded GPX for viewing; offline ObjectBox cache read-back) — not part of any of the 7 plans' `files_modified` or `must_haves`. The new module's own doc comment overclaims ("later plans redirect every existing call site") but no plan in this phase actually scoped that redirect. Real defect, real broken promise, but not a failure of a phase-34 must-have or roadmap SC — downgraded from the reviewer's implicit "critical" framing to a warning for this phase's scope. |
| CR-04 | The endpoint reflects arbitrary unauthenticated bytes verbatim with an active XML content-type, no validation | **CONFIRMED — real, phase-introduced, BLOCKER** | Diffed current `+server.ts` against the pre-phase version at commit `ca063023`: the old handler wrapped `gpx2trail(gpxData, ...)` in try/catch and threw a 400 "Invalid GPX content" on unparseable input — the *only* content validation that existed. That call and its catch are gone; `convert.test.ts:36-47` now explicitly asserts verbatim echo as intended behavior, and no test asserts rejection of malformed content. Verified the endpoint is unauthenticated (`authorization_util.ts`: `/api/v1/trail/convert` absent from `privateRoutes`) and `/api/v1` is CSRF-exempt (`hooks.server.ts:196`) — both pre-existing, not new — but the *capability* to get verbatim attacker content reflected back (as opposed to a computed, transformed Trail JSON) is new to this phase's change. |

### Additional Warnings Independently Spot-Checked (not phase-blocking)

| ID | Finding | Verified? |
|---|---|---|
| WR-01 | Off-by-one in `segmentPolylinesFromTrack`'s not-found fallback drops one polyline | Confirmed by trace: `polylines` gets `anchors.length - 2` entries instead of `anchors.length - 1` when the `idx == -1` fallback fires. |
| WR-04 | `_onFinish`'s double-tap guard (`_finishing`) is set only after `await resolveTrackSaveOptions`, not before | Confirmed by reading `route_planner_screen.dart:513-533` — the non-edit branch's `setState(() => _finishing = true)` follows the await. |
| WR-08 | Zero-length recording persists `moving_duration = 0` instead of `null` | Confirmed: `navigation_screen.dart:865` passes `movingDuration: navStats.elapsed` unconditionally; `form_data_util.dart:34` guards on `!= null`, not `> 0`. |
| WR-09 | `moving_duration` documented in OpenAPI but stripped by `TrailCreateSchema`/`TrailUpdateSchema` (Zod) | Confirmed: `openapi_schemas.ts` has 3 `moving_duration` mentions; `trail_schema.ts`'s two Zod schemas have no `moving_duration` field, so the JSON API route silently drops it (multipart `/trail/form` — the app's actual path — is unaffected). |
| WR-06 | TS/Dart diverge on a non-empty, unparseable `<time>` (TS → `NaN` duration; Dart → 0) | Confirmed: `waypoint.ts:56-58` still does `if (object.time) this.time = new Date(object.time)` with no `isNaN` guard; no corpus fixture exercises this case. |

## Required Artifacts (spot-checked against PLAN frontmatter)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `app/lib/util/gpx_conversion_util.dart` | `sanitizeGpxEleAndTime`, `parseGpxSafely`, `GpxMetricsComputation`, `trailFromGpx`, `computeTrailMetrics`, `finalElevationGain` | ✓ EXISTS, SUBSTANTIVE, WIRED | 421+ lines; wired into `elevation_profile.dart`, `trail_panel.dart`, `trail_import_util.dart`, `route_planner_handoff_util.dart` |
| `fixtures/gpx-corpus/` (10 fixtures + README) | On-disk, language-neutral corpus | ✓ EXISTS, SUBSTANTIVE, WIRED | Read by both `gpx-corpus.test.ts` and `gpx_corpus_test.dart`; verified via passing test runs |
| `db/migrations/1785300000_add_moving_duration_to_trails.go` | Additive, idempotent `moving_duration` field | ✓ EXISTS, SUBSTANTIVE | `GetByName == nil` guard present; `Down` calls `RemoveByName` |
| `app/lib/util/format_util.dart` | `trailDisplayDuration` | ✓ EXISTS, SUBSTANTIVE, WIRED | Wired at `trail_card.dart:383`, `trail_list_item.dart:364`, `trail_panel.dart:48` |
| `web/src/routes/api/v1/trail/convert/+server.ts` | Transcode-only, `application/gpx+xml` | ✓ EXISTS, SUBSTANTIVE | Functionally transcode-only; validation regression noted under Truth 4 gap |
| `app/lib/util/trail_import_util.dart` | `transcodeToGpx`, `buildLocalTrail` | ✓ EXISTS, SUBSTANTIVE, WIRED | Data-loss defect noted under Truth 1 gap |
| `app/lib/util/track_save_options_util.dart` | `resolveTrackSaveOptions` (D-15 online gate) | ✓ EXISTS, SUBSTANTIVE, WIRED | Wired at all 3 call sites (`navigation_screen.dart:736`, `route_planner_screen.dart:522`, `trail_import_util.dart:84`) |

## Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `gpx_conversion_util.dart` | `package:gpx GpxReader` | `parseGpxSafely` chains both sanitize passes | ✓ WIRED (but sanitize coverage incomplete — see CR-02) |
| `db/migrations` | `trails` collection | `moving_duration` field add | ✓ WIRED |
| `app/lib/util/form_data_util.dart` | `PUT/POST /trail/form` | multipart `moving_duration` field | ✓ WIRED |
| `app/test/util/gpx_corpus_test.dart` | `fixtures/gpx-corpus/` | relative `dart:io` File read | ✓ WIRED |
| `app/lib/components/trail/elevation_profile.dart`, `trail_panel.dart` | `computeTrailMetrics` | unsaved-GPX preview | ✓ WIRED (D-17 confirmed: `GpxMappingUtils.getTotals()`/`GpxStats` deleted from `gpx_util.dart`) |
| `route_planner_screen.dart` `_onFinish`, `trail_import_util.dart` `importTrailFile`, `navigation_screen.dart` `_saveRecordedTrack` | `resolveTrackSaveOptions` | online gate ahead of handoff | ✓ WIRED (all 3) |
| `trail_import_util.dart` `buildLocalTrail` | `trailFromGpx` | local conversion replacing HTTP POST | ✓ WIRED |
| `trail_import_util.dart` `transcodeToGpx` | `POST /api/v1/trail/convert` | raw GPX response consumed on-device | ✓ WIRED, but the endpoint side of this link now has no input validation (CR-04) |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| TS corpus suite passes | `npx vitest run src/lib/models/gpx/gpx-corpus.test.ts` | 30/30 pass | ✓ PASS |
| Full web test suite | `npx vitest run` | 91/91 pass (9 files) | ✓ PASS |
| svelte-check | `npx svelte-check --tsconfig ./tsconfig.json` | 0 errors, 0 warnings | ✓ PASS |
| Dart corpus suite passes | `flutter test test/util/gpx_corpus_test.dart` | 30/30 pass | ✓ PASS |
| Full Flutter test suite | `flutter test` | 580 pass, 1 skip, 4 fail (the documented pre-existing `settings_tab_test.dart` failures — confirmed identical set, no new failures) | ✓ PASS (matches known baseline) |
| Independent haversine recompute (fixture 01) | Python, `R=6371000`, standard haversine | `134.59240148587796` | ✓ PASS — bit-identical to `expected.json` and to the reviewer's claim |
| PORT-03 gate | `grep -rn "trail/convert" app/lib/` | 1 match (`trail_import_util.dart:175`, inside `transcodeToGpx`) | ✓ PASS |

## Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` probes declared or discovered for this phase; verification relied on the project's standard Vitest/Flutter test suites and direct source review instead.

## Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the 16 files reviewed. No stub returns or empty handlers found in the reviewed conversion/import/endpoint code. The defects found (CR-01, CR-02, CR-04) are logic gaps, not stub markers — which is exactly why they survived `flutter analyze`/`svelte-check`/test-suite green states: the missing behavior has no test asserting it, so nothing fails loudly.

## Human Verification Required

None identified — all findings in this report were verified programmatically (source read, package source read, git diff against pre-phase commit, and live test-suite execution). No visual, real-time, or external-service-dependent behavior required human judgment for this report's conclusions.

## Gaps Summary

The phase delivers a genuinely faithful numeric port (independently re-verified, not just trusted from the SUMMARYs or the code review) and correctly wires all three capture paths off the server round trip, closing PORT-02, PORT-03, and CONV-06 cleanly. It fails the phase goal on two fronts that are squarely inside "computes ... from a GPX ... entirely on-device ... for recordings, route-planner output, and file imports" and "the server's convert endpoint stops computing trails at all [safely]":

1. **Data loss on corrected imports (CR-01):** any file import where the user accepts the "recalculate heights" or "follow roads" option (both now offered by this phase's own D-15 UI extension to the import path) permanently strips the file's own waypoints, name, and description — the exact opposite of "the app computes a trail's name, waypoints ... from a GPX."
2. **Import crashes on common real-world GPX (CR-02):** the sanitizer this phase introduces as "the single sanctioned parse entry point" only guards 2 of the ~10 numerically-parsed GPX tags, so files from common GPS loggers/exporters that previously imported successfully (via the old server-side `xml2js` path, which never threw) now fail outright.
3. **A new unauthenticated content-reflection defect on the convert endpoint (CR-04):** removing the endpoint's only validation, with no replacement, turns it into a verbatim echo of arbitrary request bodies under an active XML content type — a regression against the pre-phase behavior where every successful response was a computed, transformed Trail JSON.

CR-03 (two pre-existing GPX parse sites not redirected through `parseGpxSafely`) is real but out of this phase's declared scope (no plan's `files_modified` or `must_haves` touches `trail_provider.dart`/`trail_entity.dart`) — recorded as a warning, not a blocker, though it should likely be folded into whichever plan closes CR-02, since fixing the sanitizer without fixing the redirect leaves those two sites exposed to the same crash class.

None of these findings are addressed by a later milestone phase's stated goal or success criteria (Phase 35 is scoped to `trail_create_screen` offline UX; Phase 36 to local-first recording/sync) — nothing here was deferred, so no items were moved to a `deferred:` section.

---

_Verified: 2026-08-01T08:45:38Z_
_Verifier: Claude (gsd-verifier)_
