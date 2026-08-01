---
phase: 34-dart-conversion-port
audited: 2026-08-01T21:15:00Z
auditor: Claude (gsd-secure-phase)
threats_declared: 39
threats_open: 0
threats_closed: 39
accepted_risks: 9
transfer_risks: 0
na_rows: 7
verification_basis: "Working tree at 1e060d3a — i.e. after the four UAT-driven fixes 7b5bfac3, 9dc5a69f, a84e7ab9 and 1a54d99d. All 7 plans' <threat_model> blocks, cross-checked against live code reads and test runs, not SUMMARY/REVIEW claims taken on faith"
---

# Phase 34: Dart Conversion Port — Security Audit

**Scope:** every threat declared in `34-01-PLAN.md` through `34-07-PLAN.md`'s `<threat_model>`
blocks (39 numbered threats, T-34-01..T-34-39, plus a repeated `T-34-SC` "no new packages" row in
each of the 7 plans). Verified against the code as it stands at `HEAD` (`c62527b2`), i.e. **after**
the four UAT-driven commits (`7b5bfac3`, `9dc5a69f`, `a84e7ab9`, `1a54d99d`) and **after** the round-3
code review's fixes (`8724e7eb`, `c62527b2`). Documentation/plan text was treated as a claim to
verify, not evidence — every row below cites a `file:line` I read directly, a grep I ran, or a test
I executed in this session.

**Result: 0 blocking gaps.** Every `mitigate`-disposition threat has a live code artifact
implementing it (several were re-implemented via a different mechanism than the plan originally
described — noted per-row — and are functionally stronger, not weaker, than declared). All
`accept`/`n/a` rows are recorded below with their rationale. Four items surfaced during this audit
that sit **outside** Phase 34's declared threat register are reported as follow-up notes, not as
Phase 34 blockers, per the audit's own scope rule ("verify declared threats, don't scan blindly").

## Threat Verification

### Plan 34-01 — Core algorithm (`gpx_conversion_util.dart`)

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-01 | DoS — `GpxReader` throws on malformed `<ele>`/`<time>` | mitigate | **VERIFIED** (mechanism superseded, stronger) | The plan's literal mitigation (`sanitizeGpxEleAndTime` regex pre-pass) no longer exists — `grep -rn "sanitizeGpxEleAndTime\|sanitizeGpxNumericAndTime" app/` returns nothing. It was superseded by vendoring the reader itself: `app/lib/vendor/gpx/gpx_reader.dart` uses `tryParse` at every numeric/date coercion site (`_readDouble`/`_readInt`/`_readDateTime`, lines 448-461) instead of the throwing `parse`. `parseGpxSafely` (`gpx_conversion_util.dart:43-45`) is the sole call site, enforced by a repo-guard test (`gpx_conversion_util_test.dart:1051-1162`, run: 136/136 tests pass). A `<trkpt>` missing `lat`/`lon` no longer throws `StateError` at all (LOCAL MODIFICATION 2) — stronger than the plan's own stated residual case. |
| T-34-02 | Tampering — non-finite haversine poisons distance | mitigate | **VERIFIED** | `haversineMeters` (`gpx_conversion_util.dart:98-105`) returns `double.nan` for a null coordinate; `_readCoordinateAttribute` (`gpx_reader.dart:427-439`) additionally rejects `NaN`/`Infinity`/`1e999` at the parse boundary (`isFinite` check, line 438) — this closes a gap a later code review (round 3, CR-01) found in the original null-only guard, now fixed. `addAndFilter` gates on `hasUsablePosition` before touching any accumulator (`gpx_conversion_util.dart:223-225`). |
| T-34-03 | Tampering — `parseGpxElevation` accepting NaN/Infinity | mitigate | **VERIFIED** | `parseGpxElevation` (`gpx_conversion_util.dart:53-58`) returns `null` unless `raw.isFinite`. Reinforced by the reader-level fix above. |
| T-34-04 | DoS — unbounded point count | accept | **CLOSED** (recorded below) | No point-count limit exists; rationale (O(n), no nested loops, GPS-sample-rate-bounded input) unchanged and still holds — `computeTrailMetrics`/`addAndFilter` are single-pass. |
| T-34-05 | Info disclosure — pure module leaking coords over network | mitigate | **VERIFIED** | `grep -nE "^import 'package:(dio\|flutter_riverpod)/\|^import 'dart:io'" app/lib/util/gpx_conversion_util.dart` → no matches (checked live). |
| T-34-SC | Supply chain | n/a | **CLOSED** (see note) | See "Supply-chain note" below — one package promoted transitive→direct with identical version/hash; no new package. |

### Plan 34-02 — `moving_duration` field

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-06 | Tampering — route mutation silently overwrites recorded moving time | mitigate | **VERIFIED** (implementation evolved beyond plan text) | The plan's original claim ("no writer of `moving_duration` exists on the web at all") is no longer literally true: `web/src/routes/trail/edit/[id]/+page.svelte:1607-1628`'s `updateTotals()` now explicitly writes `moving_duration: 0` on every route mutation. This is a **deliberate, later fix** (WR-10, commit `7d393431`) closing a real data-integrity gap the original "never touch it" design left open (a stale `moving_duration` from a previous, now-replaced route would otherwise survive and be shown in preference to the freshly recomputed `duration`). The field is cleared, never set to a fabricated non-zero value — the underlying threat (spurious moving time attributed to a route it doesn't describe) is mitigated, more robustly than originally planned. |
| T-34-07 | Tampering — negative/absurd `moving_duration` | mitigate | **VERIFIED** | Migration sets `Min: &movingDurationMin` (`float64(0)`) at `db/migrations/1785300000_add_moving_duration_to_trails.go:32`. Additionally hardened beyond the plan: `web/src/lib/models/api/trail_schema.ts:25,56` now validates `moving_duration: z.number({coerce:true}).nonnegative().optional()` on both `TrailCreateSchema`/`TrailUpdateSchema` (WR-09 fix, closing a gap where the OpenAPI-documented field was silently stripped by the Zod layer). |
| T-34-08 | Tampering — migration data loss on production | mitigate | **VERIFIED** | `db/migrations/1785300000_add_moving_duration_to_trails.go:27-40`: `Up` only calls `Fields.Add` behind `GetByName("moving_duration") == nil`; `Down` (lines 43-49) calls `RemoveByName` only. `go build ./...` clean. |
| T-34-09 | Tampering — ObjectBox property id reuse | mitigate | **VERIFIED** | `git diff fb381452..HEAD -- app/lib/objectbox-model.json` shows only additions (`"name": "movingDuration"` with a fresh id), no existing id changed. |
| T-34-10 | Info disclosure — aggregate duration on a public trail | accept | **CLOSED** (recorded below) | Rationale unchanged: a single aggregate strictly less revealing than the already-public per-point GPX timestamps. |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages (build_runner/npm run openapi:generate are pre-existing dev deps). |

### Plan 34-03 — Shared GPX corpus (test-only surface)

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-11 | Tampering — fixture edited/deleted, parity suite silently narrows | mitigate | **VERIFIED** (floor raised) | Corpus now has 12 fixtures (was 10 at plan time); both suites assert a floor: `app/test/util/gpx_corpus_test.dart:72` (`if (fixtures.length < 12)`), `web/.../gpx-corpus.test.ts:94` (`toBeGreaterThanOrEqual(12)`) — both re-verified live. |
| T-34-12 | Tampering — oracle contamination (expected values derived by running the implementation) | mitigate | **VERIFIED** | `ls fixtures/gpx-corpus/*/DERIVATION.md \| wc -l` → 10 (grew from 9 with the WR-06 fixture 11 addition); each states its derivation is independent of the implementation under test. Fixture 12 (`12-dense-switchback`) added later by the quick-260801-opr task, also present with its own derivation artifact. |
| T-34-13 | DoS — path traversal/unbounded read via corpus reader | accept | **CLOSED** | Test-only code; `readdirSync`/`Directory('../fixtures/gpx-corpus')` on a fixed repo-relative path, no user input. |
| T-34-14 | Tampering — false green (corpus resolves to empty/wrong dir) | mitigate | **VERIFIED** | `existsSync`/`Directory.current.path` guards present in both suites (checked live, both pass). |
| T-34-15 | DoS — malformed-XML fixture hangs the parser | accept | **CLOSED** | TS parser (`isomorphic-xml2js`) handles it as `undefined`; Dart-side crash risk is what T-34-01's mitigation (now the vendored reader) covers. |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages. |

### Plan 34-04 — Dart `trailFromGpx` + corpus proof + deleting the 2nd metrics impl

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-16 | DoS — `trkpt` missing `lat`/`lon` throwing `StateError` inside a caller | mitigate | **VERIFIED** (stronger than declared) | `trailFromGpx` never force-unwraps a coordinate (uses `hasUsablePosition`/`firstWhereOrNull`, `gpx_conversion_util.dart:551-553`). The residual `StateError` case the plan named ("still throws... handled by existing try/catch") **no longer exists at all**: LOCAL MODIFICATION 2 in the vendored reader tolerates missing/unparseable `lat`/`lon` unconditionally, so there is no exception path left to catch. |
| T-34-17 | Tampering — stored injection via GPX `name`/`description` text | accept | **PARTIALLY MISCALIBRATED rationale, not a Phase-34 regression — see "Rendering rationale" note below** | The field flow claim ("these exact fields already flow from GPX text via `gpx2trail` today") is correct and unchanged by Phase 34. The claim "rendering layers... escape by default" is **not accurate** for the description field specifically — see finding below. Not a Phase 34 blocker (pre-existing sink, not introduced or widened by this phase), but the accepted-risk *rationale* should be corrected. |
| T-34-18 | Tampering — icon spoofing via `sym` | mitigate | **VERIFIED** | `fontAwesomeIconsMap[wpt.sym] ?? FontAwesomeIcons.circle` (`gpx_conversion_util.dart:540`, and again at `trailFromGpx`'s waypoint loop, line ~527-544). `grep -c "icon: wpt.sym"` → 0. |
| T-34-19 | Tampering — two disagreeing metrics implementations | mitigate | **VERIFIED** | `grep -c "getTotals\|class GpxStats" app/lib/util/gpx_util.dart` → 0; `grep -rn "getTotals" app/lib/` → no matches at all (stronger than the plan's own acceptance criterion, which allowed a comment-only false positive that was since removed). `elevation_profile.dart`/`trail_panel.dart` both call `computeTrailMetrics`. |
| T-34-20 | Tampering — false green Dart corpus suite | mitigate | **VERIFIED** | Same `existsSync` + floor guard as T-34-14, Dart side (`app/test/util/gpx_corpus_test.dart`). |
| T-34-21 | Info disclosure — pure module network leak | mitigate | **VERIFIED** | Same D-14 import grep as T-34-05; `flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart` → 136 tests, all pass (run live). |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages. |

### Plan 34-05 — Switch capture paths to Dart conversion

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-22 | DoS — malformed `.gpx` parsed on-device crashing import | mitigate | **VERIFIED** | `importTrailFile` (`trail_import_util.dart:82-193`) wraps the whole parse+build in `try { } catch (e, st) { showError(); }`; `parseGpxSafely` used at line 87. |
| T-34-23 | DoS — large file read fully into memory | accept | **CLOSED** | `File(path).readAsString()` (line 84) has the identical memory profile as the previously-used `MultipartFile.fromFile` on the same path; unchanged, user-selected file. |
| T-34-24 | Info disclosure — geocode coordinates sent without intent | mitigate | **VERIFIED** | `buildLocalTrail` (`trail_import_util.dart:251-284`) gates the reverse-geocode call on `ref.read(onlineStatusProvider) && lat != null && lon != null` (line 267), targets the app's own proxy, `includeRoad: false` matches old server behaviour. |
| T-34-25 | Tampering — transcode shim silently produces an empty trail | mitigate | **VERIFIED** | `transcodeToGpx` (`trail_import_util.dart:214-236`) throws `StateError` unless one of the two response shapes yields a non-empty string (lines 222-235). |
| T-34-26 | Tampering — split-brain `navigationStatsProvider` seed | mitigate | **VERIFIED** | `grep -n "navigationStatsProvider(widget.response, resume: _resumeStats)" app/lib/routes/navigation_screen.dart` → 4 identical occurrences (lines 640, 643, 758, 1124), one inside `_saveRecordedTrack` (checked live). |
| T-34-27 | Repudiation/data loss — offline save failure losing the session | mitigate | **VERIFIED** | `active_nav.clear(_store)` (`navigation_screen.dart:878`) runs only *after* `buildDraftTrail` succeeds (line 870-876); the `catch` block (884-894) never reaches the clear call. |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages. |

### Plan 34-06 — Extend save-options gate; fix offline planner dead end

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-28 | Tampering — snapped planner leg drifting off anchors | mitigate → **superseded by feature removal** | **CLOSED** (attack surface removed) | The planner's save-options sheet (and its `snapCosting`/road-snap block) was **deleted entirely** in the UAT fix `7b5bfac3` — the planner never road-snaps at all now (`buildFinalPlannedGpx(WidgetRef ref)`, `route_planner_handoff_util.dart:436`, takes no snap params; `grep -n "snapCosting\|refetchAllHeights"` in this file → 0 matches). The anchor-drift risk this mitigation existed for no longer has a code path to trigger it. Verified per the UAT resolution: the user reported the sheet had "no purpose" for the planner (route already Valhalla-routed leg-by-leg), and diagnosis confirmed the toggle was provably inert/redundant. |
| T-34-29 | Tampering — partial Valhalla map-match truncating track | mitigate | **VERIFIED** | `snapResultAcceptable` (`route_planner_handoff_util.dart:195-213`) still gates the recording and import paths' `snapShapeToRoads`/`snapShapeToRoadsResult` (lines 229-260+). Unchanged, still reachable there. |
| T-34-30 | Tampering — import saves a transformed track whose `gpxData` still holds the untransformed original | mitigate | **VERIFIED** | `trail_import_util.dart:174`: `finalGpxData = GpxWriter().asString(finalGpx);` only when a transform ran (`recalcHeights \|\| followRoads`, line 121); untransformed path passes the original text (`finalGpxData = gpxXml` at line 119, unchanged when neither toggle fires). |
| T-34-31 | DoS — optimistic `onlineStatusProvider` causing a bounded stall | accept | **CLOSED** | Unchanged: best-effort transforms with silent fallback, bounded by `connectTimeout`; same posture the recording path shipped with previously. |
| T-34-32 | Info disclosure — full-res coords sent to Valhalla proxy | mitigate | **VERIFIED, with a scoping caveat** | Toggle off by default, never shown offline, confirmed. The claim "the endpoints are the app's own authenticated proxies" is accurate for `/valhalla/trace-route` (`event.locals.user` check, `trace-route/+server.ts:73-75`) but **not** for `/valhalla/height` (no `locals.user` check in that handler) — see finding below. Both endpoints predate Phase 34 (`git diff fb381452..HEAD` on both files is empty) and neither was modified by this phase, so this is a scoping nuance in the threat-model prose, not a Phase 34 regression. |
| T-34-33 | Repudiation/data loss — cancelled sheet mistaken for an error | mitigate | **VERIFIED** | All three call sites (`navigation_screen.dart:741`, `route_planner_screen.dart` gate removed per T-34-28 note above, `trail_import_util.dart:109`) return early on `options == null` with no toast; session state untouched. |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages. |

### Plan 34-07 — Convert endpoint transcode-only

| Threat ID | Category | Disposition | Verdict | Evidence |
|-----------|----------|-------------|---------|----------|
| T-34-34 | DoS — malformed upload crashing/hanging transcode | mitigate | **VERIFIED, strengthened** | Whole handler inside `try/catch` → `handleError` (`+server.ts:78-138`); empty-body 400 guard intact (line 115-117). Strengthened beyond the original plan: `assertParsableGpx` (lines 24-39) now validates every input branch with `GPX.parse` before any 200 response — this was added later (CR-04 fix, commit `38c6db7b`) after a code-review finding that the transcode-only rewrite had **temporarily dropped all content validation**, making the endpoint reflect arbitrary unauthenticated bytes under an XML content type. That gap is now closed; see finding below for the interim-gap note. |
| T-34-35 | Info disclosure — response echoing more than the transcoded document | mitigate | **VERIFIED** | Success path returns exactly `gpxData` (line 124), no trail/user data. `convert.test.ts`'s D-06 regression guard (lines 178-192) asserts the 200 body is not JSON and contains neither `expand` nor `elevation_gain` — run live, passes. |
| T-34-36 | Tampering — OpenAPI JSON drifting from actual contract | mitigate | **VERIFIED (partially, by design)** | `web/static/docs/api/wanderer.openapi.json` is gitignored (`web/.gitignore:6`) and not committed — this is stated as deliberate in `34-07-SUMMARY.md` (build artifact, regenerated via `npm run openapi:generate`), not a gap. The generating source (`+server.ts`'s `@swagger` JSDoc, lines 41-77) was directly read and correctly describes the `application/gpx+xml` / `type: string` response with no `Trail` reference. |
| T-34-37 | Repudiation/availability — older app build breaking on non-GPX import | accept | **CLOSED** | Rationale unchanged and accurate: narrow blast radius, fails loudly, `transcodeToGpx`'s dual-shape tolerance (verified at `trail_import_util.dart:222-235`) means only the old-app/new-server direction breaks. |
| T-34-38 | Tampering — JSON error body mistaken for GPX | mitigate | **VERIFIED** | Errors keep the `handleError` JSON shape + non-200 status (line 136); `transcodeToGpx` only reads a body on success and throws `StateError` otherwise (confirmed above, T-34-25). |
| T-34-39 | DoS — unbounded upload to an unauthenticated endpoint | accept | **CLOSED** | Confirmed unchanged: no body-size limit was added or removed by this phase; `git diff fb381452..HEAD` shows this posture untouched. |
| T-34-SC | Supply chain | n/a | **CLOSED** | No new packages. |

## CR-04: an interim gap worth recording explicitly

Between the Phase-34-07 commit that made the convert endpoint transcode-only (`df61d581`) and the
code-review fix that added `assertParsableGpx` (`38c6db7b`), the endpoint had **no content
validation at all** — an unauthenticated, CSRF-exempt route (`/api/v1` is CSRF-exempt per
`hooks.server.ts`) that reflected any submitted bytes verbatim under `Content-Type:
application/gpx+xml`, which browsers parse as XML (XSLT processing instructions, XHTML-namespace
script). This was found by code review before shipping and is now closed
(`web/src/routes/api/v1/trail/convert/+server.ts:24-39`, `assertParsableGpx`, plus
`X-Content-Type-Options: nosniff` and `Content-Disposition: attachment` at lines 131-132). Verified
live: `npx vitest run src/routes/api/v1/trail/convert/convert.test.ts` → 14/14 pass, including 7
rejection-case tests (plain text, HTML, non-GPX XML, XHTML+script, malformed XML, JSON blob, and
the JSON-content-type variant) each asserting the response does not echo the submitted body. **No
action needed — recorded here because the gap was real and the fix is what makes T-34-34/35/38
true at HEAD, not the original plan text.**

## Findings outside Phase 34's declared threat register

These surfaced during this audit's verification pass. None are Phase-34-introduced regressions
(each file involved is either byte-unchanged since the Phase 33 baseline `fb381452`, or the
concern predates this phase's design). Reported per the audit brief's explicit ask to check these
specific surfaces; classified as **WARNING / unregistered_flag**, not BLOCKER.

1. **`{@html}` rendering of `trail.description`/waypoint `description` (web) and `Html(data:
   trail.description)` (Flutter) render attacker-influenceable GPX `<desc>` text as markup, not
   escaped text.** `web/src/lib/components/trail/trail_info_panel.svelte:833` and
   `trail_timeline.svelte:91` use `{@html}` directly on `trail.description`/`wp.description` with
   no `marked()`/DOMPurify call found anywhere in the surrounding code (`grep -rln "DOMPurify" web/src`
   → no matches). `app/lib/components/trail/trail_panel.dart:267` renders the same field via
   `flutter_html`'s `Html(data: ...)` (no JS execution risk there, but still raw-markup rendering).
   T-34-17's accept-disposition rationale ("rendering layers... escape by default") does not hold
   for this specific sink. **Not introduced by Phase 34** — `git diff fb381452..HEAD` on both
   `.svelte` files shows the `{@html}` lines untouched; the field has flowed from GPX text into
   `Trail.description` via the pre-existing server-side `gpx2trail` since before this phase. Phase
   34 does not widen this: it moves *where* the same field is computed (client vs. server), not
   *what* is trusted. Recommend re-stating T-34-17's accepted-risk rationale accurately (the
   correct justification is "no new sink, no new source, unchanged trust boundary" — not
   "escapes by default") and opening a follow-up item to sanitize `description` before render on
   both platforms.
2. **`/api/v1/valhalla/height` has no explicit auth check**, unlike `/api/v1/valhalla/trace-route`
   (`event.locals.user` gate, `trace-route/+server.ts:73-75`). `height/+server.ts` has no such
   check, and it is not listed in `authorization_util.ts`'s `privateRoutes`, so on a non-"private
   instance" deployment it is reachable unauthenticated. Confirmed pre-existing and untouched by
   Phase 34 (`git diff fb381452..HEAD` on this file is empty). T-34-32's text only claims
   `trace-route` is authenticated, which is accurate — the nuance is that a reader could
   over-generalise "the endpoints" (plural) as both being gated. Not a Phase 34 regression;
   worth a follow-up ticket regardless, since `/height` still proxies arbitrary
   attacker-supplied coordinate arrays to an internal service with no request-size cap that was
   found in this read.
3. **KMZ zip decompression (`fromKMZ`, `web/src/lib/util/gpx_util.ts:192-197`) has no
   decompression-size limit** (`JSZip.loadAsync` on the full archive, no zip-bomb guard). Reads a
   single fixed filename (`doc.kml`) so there is no path-traversal risk (nothing is extracted to
   the filesystem; it's an in-memory read by a hardcoded name). This file is confirmed
   byte-unchanged since the Phase 33 baseline (`git diff fb381452..HEAD -- web/src/lib/util/gpx_util.ts`
   is empty) and is explicitly out of this phase's scope (34-07-PLAN.md's own instructions:
   "Do not touch... `web/src/lib/util/gpx_util.ts`"). Not a Phase 34 finding; flagged only because
   the audit brief named KMZ handling explicitly as a surface to check.
4. **Supply-chain note (T-34-SC accuracy):** `app/pubspec.yaml` gained a new direct-dependency
   line for `xml: ^6.3.0` (`git diff fb381452..HEAD -- app/pubspec.yaml`). `pubspec.lock` shows
   this is the exact same package/version/sha256 that was already resolved as a **transitive**
   dependency of `package:gpx` before this phase — promoted to direct because the vendored
   `gpx_reader.dart` now imports `package:xml/xml_events.dart` directly (line 51). No new
   supply-chain surface (same already-resolved artifact), but T-34-SC's literal "no new packages"
   claim is not quite exact — a new manifest line did appear. Purely a documentation-precision
   note, not a security gap.

## Accepted Risks Log

| ID | Risk | Rationale | Status |
|----|------|-----------|--------|
| T-34-04 | Unbounded GPX point count → CPU/battery cost | O(n) single-pass computation, no nested loops; real-world GPX bounded by GPS sample rate; no limit exists in code | Accepted, unchanged |
| T-34-10 | `moving_duration` reveals aggregate pause behaviour on a public trail | Strictly less revealing than the already-public per-point GPX timestamps | Accepted, unchanged |
| T-34-13 | Corpus reader path handling | Test-only code, fixed repo-relative path, no user input | Accepted, unchanged |
| T-34-15 | Malformed-XML fixture parser behaviour (TS side) | Existing TS parser tolerance; Dart-side risk covered by T-34-01's mitigation | Accepted, unchanged |
| T-34-17 | Stored injection via GPX `name`/`description` text | Pre-existing trust posture unchanged by this phase — **see Finding 1 above; rationale text should be corrected, but disposition (accept, unchanged risk) is not wrong** | Accepted, rationale needs correction (non-blocking) |
| T-34-23 | Whole-file read into memory on import | Identical memory profile to the pre-existing `MultipartFile.fromFile` path | Accepted, unchanged |
| T-34-31 | Optimistic `onlineStatusProvider` causing a bounded stall before falling back | Best-effort with silent fallback, bounded by client `connectTimeout`; unchanged posture | Accepted, unchanged |
| T-34-37 | Older app build breaks on non-GPX import after the hard contract change | Narrow blast radius, fails loudly, self-hosted deployment model; `transcodeToGpx` dual-shape tolerance limits blast to one direction | Accepted, unchanged |
| T-34-39 | Unbounded upload size to an unauthenticated transcode endpoint | Unchanged posture from before this phase; out of this phase's declared scope | Accepted, unchanged |

## Test Evidence (run live in this audit session)

| Command | Result |
|---------|--------|
| `cd web && npx vitest run src/routes/api/v1/trail/convert/convert.test.ts` | 14/14 pass |
| `cd web && npx vitest run` (full suite) | 115/115 pass, 10 files |
| `cd app && flutter test test/util/gpx_conversion_util_test.dart test/util/gpx_corpus_test.dart` | 136/136 pass |
| `cd app && flutter test` (full suite) | matches documented pre-existing baseline exactly — only the 4 known-unrelated `settings_tab_test.dart` failures, no new failures |
| `grep -nE "^import 'package:(dio\|flutter_riverpod)/\|^import 'dart:io'" app/lib/util/gpx_conversion_util.dart` | no matches (D-14 purity holds) |
| `grep -rn "getTotals" app/lib/` | no matches |
| `grep -rn "trail/convert" app/lib/` | exactly 1, inside `transcodeToGpx` |

## Unregistered Flags (SUMMARY.md `## Threat Flags`)

None of the 7 plans' SUMMARY.md files contain a `## Threat Flags` section — no executor-flagged
new attack surface to reconcile.

## Conclusion

**threats_open: 0.** All 39 declared threats (plus the 7 recurring `T-34-SC` "no new packages"
rows) resolve to CLOSED — either `mitigate` with live code evidence (several strengthened by
post-plan code review and UAT fixes), or `accept`/`n/a` with an unchanged, still-valid rationale.
One threat's accepted-risk *rationale* (T-34-17) should be corrected to match the actual rendering
code, and four out-of-scope observations are recorded as follow-up candidates, none blocking this
phase.
