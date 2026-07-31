# Phase 34: Dart Conversion Port - Context

**Gathered:** 2026-07-31
**Status:** Ready for planning

<domain>
## Phase Boundary

The app derives a trail's name, description, waypoints, start coordinates, date, distance,
elevation gain/loss, duration, and bounding box from a GPX **entirely on-device with no
network call** — for recordings, route-planner output, and file imports — proven identical
to the Phase 33-corrected web implementation by a shared fixture corpus. The server's
`POST /api/v1/trail/convert` stops computing trails and becomes transcode-only.

**In scope:** PORT-01..05, CONV-06. Dart conversion module + tests; the shared fixture
corpus and its TS-side reader; the convert endpoint's contract change and OpenAPI update;
switching the app's three capture paths onto the Dart path; the `moving_duration` schema
addition; extending `showTrackSaveOptionsSheet` to the planner and import paths.

**Out of scope:** The two `trail_create_screen` offline gaps (blank map, throwing tag
autocomplete) — those are Phase 35. Any change to the Phase 33 TS computation itself
beyond what the corpus reader needs.

</domain>

<decisions>
## Implementation Decisions

### Shared fixture corpus (PORT-02)

- **D-01 [locked]:** The corpus lives **on disk in a language-neutral format** (GPX input +
  expected-metrics file per fixture), read by both the Vitest suite and the Dart tests.
  This **amends Phase 33's D-05** ("fixtures are inline, not disk files") to: *inline for
  TS-only unit tests, on-disk for the cross-language contract*. A single source of truth is
  the whole point of PORT-02 — duplicated per-language fixtures can drift, and a drifted
  fixture would make PORT-02 pass while the implementations disagree.

- **D-02 [locked]:** Expected values are **hand-derived from first principles for the
  CONV-01..05 defect cases**, with the derivation documented alongside each fixture (these
  are small known-answer cases — the 88 m scramble, the stationary ±7 m oscillation → 0/0,
  the 2-point segment). Bulk realistic-track fixtures may be **seeded from the corrected TS
  output and human-reviewed** before locking. The TS implementation is explicitly **not** the
  oracle — treating it as one would bake any surviving TS bug in as "expected", the exact
  failure mode Phase 33's sequencing exists to prevent.

- **D-03 [locked]:** The corpus asserts a **tight absolute tolerance, documented per field**:
  distance and elevation to ~1e-6 m; duration, point counts, and bounding box exact. Dart and
  JS are both IEEE 754, but `dart:math` and V8 trig are not required to agree bit-for-bit, so
  exact equality on a haversine sum could fail for reasons that are not defects — and such a
  failure would be very hard to distinguish from a real one. 1e-6 m still catches every
  algorithmic divergence.

- **D-04 [locked]:** The corpus asserts **public metrics only** — distance, elevationGain/Loss,
  duration, boundingBox, centroid, waypoints, name, description. Dart internals may differ.
  **`cumulativeDistance` is NOT ported**: its only consumer is the web crop slider (Phase 33
  D-01/D-02), which has no app equivalent, so porting it would be dead Dart code. The
  monotonic-vs-final elevation split is likewise not required of Dart — but note the port must
  reproduce the *values* `finalElevationGain`/`finalElevationLoss` produce (they include a
  still-pending noise excursion), not the values `totalElevationGainSmoothed`/`LossSmoothed`
  produce. Porting the wrong one of the pair is the single most likely way to fail the corpus.

### Convert endpoint (PORT-04)

- **D-05 [locked]:** **Hard break** — change the response in place, no `/v2` and no content
  negotiation. Old app builds calling it will break on kml/kmz/tcx/fit import. Accepted
  because the blast radius is narrow (non-GPX import only, old builds only), it fails loudly
  at import time rather than corrupting data, and self-hosters control their own server/app
  pairing. Versioning or negotiating would keep the legacy trail-computing path alive on the
  server, directly contradicting the phase goal.

- **D-06 [locked]:** Success response is the **raw GPX document** with
  `Content-Type: application/gpx+xml`. Errors keep the existing JSON shape via `handleError`,
  so the app's error handling is unchanged. The multipart and JSON/raw-text *input* branches
  stay as they are — the app's existing multipart upload at `trail_import_util.dart:83` must
  keep working.

- **D-07 [locked]:** The reverse-geocode step the endpoint performs today
  (`+server.ts:87-99`, filling `trail.location` via Nominatim) **moves to the app** as a
  separate, optional call made only when online. Offline, `location` is simply empty and the
  user types it. This keeps computation and geocoding as separate concerns and lines up with
  Phases 35/36, where a place name genuinely requires network.

- **D-08 [locked]:** The **web frontend is unchanged**. It already computes client-side via
  `gpx2trail` against the Phase 33 code and only uses the endpoint for kml/kmz/tcx/fit
  transcoding — which is exactly what the endpoint still does. Both clients converge on
  "server transcodes, client computes" with no web regression risk.

### Moving time (CONV-06)

- **D-09 [locked]:** **No new derivation is needed.** `NavigationStats.elapsed` is already
  moving time: `navigation_stats_provider.dart`'s 1-second tick is a no-op while
  `isPaused || isStationary`, so `_pausedAccum` already absorbs both manual pauses and
  tracelet's native stationary detection. It is also already the value displayed to the user
  during a session. The trail's moving time is the session's final `elapsed`.

- **D-10 [locked]:** Moving time is stored in a **new, separate `moving_duration` field**;
  `duration` keeps exactly one meaning everywhere — GPX-derived elapsed (last trkpt time
  minus first). Display rule: show `moving_duration` when present, else `duration`.

  **Rationale (this is the load-bearing decision of the phase):** ~14 call sites in
  `web/src/routes/trail/edit/[id]/+page.svelte` funnel through `updateTrailWithRouteData()`
  → `updateTotals(valhallaStore.route)`, which recomputes all four metrics from the GPX.
  The load path does *not* recompute (verified: it only does `setRoute` → `initRouteAnchors`
  → `updateTrailOnMap`), so editing name/description is safe — but any route mutation
  (nudge an anchor, undo/redo, crop, reverse, reset) would overwrite a recording's moving
  time with GPX elapsed. Moving time is not recoverable from a GPX, so that loss is
  **silent, one-way, and permanent**, triggered by an innocuous action. There is also **no
  provenance field on `Trail`** in either the TS or Dart model (verified), so the web has no
  way to know `duration` ever meant something else. A separate field makes `updateTotals`
  structurally incapable of destroying it, rather than relying on every future writer of
  `duration` to remember a guard — the precise failure mode Phase 33 hit three times.

- **D-11 [locked]:** A recording takes **only `moving_duration`** from the session. Distance,
  elevation gain/loss, and `duration` all come from the ported Dart computation over the
  recorded GPX. This refines the user's initial framing ("the recording provides the trail's
  stats") down to the one value that genuinely cannot be derived from a GPX. Consequence:
  every persisted stat is reproducible, matches what a later web recompute produces, and is
  covered by the fixture corpus — so no stat silently shifts on a first web edit.

- **D-12:** CONV-06 **cannot be pinned by the shared corpus** (moving time is not a function
  of a GPX — the same GPX legitimately yields different values by provenance). It needs its
  own Dart-side test on the session → trail hand-off.

- **D-13:** The Dart conversion accepts an **optional duration override** so one conversion
  path serves all three sources: GPX-derived elapsed when absent, session value when passed.

### Elevation correction / online-only options

- **D-14 [locked]:** The Dart conversion is **pure and offline by construction** — GPX in,
  metrics out, no network. Elevation correction and road-snapping are **caller-driven steps
  outside it**: fetch corrected heights / map-match, write into the GPX, then re-run the same
  pure conversion.

- **D-15 [locked]:** `showTrackSaveOptionsSheet`
  (`app/lib/components/navigation/track_save_options_sheet.dart`, returns
  `(bool recalcHeights, bool followRoads)?`, **both off by default**) becomes the single
  post-capture gate for **all three** sources. It is shown **only when online**; offline the
  app **skips it and goes straight to `trail_create_screen`**.

  | Source | Today | After Phase 34 |
  |---|---|---|
  | Recording | sheet already shown (`navigation_screen.dart:733`) | unchanged |
  | Planner | corrects unconditionally; **strands the user offline** | sheet when online; offline → create screen |
  | File import | never corrects | sheet when online; offline → create screen |

  Because both toggles default to off, the offline path is behaviourally identical to "user
  declined both" — one code path, not two. `followRoads` is Valhalla map-matching, so it is
  equally online-only and the same connectivity gate covers both.

- **D-16 [locked]:** This fixes a real offline defect in passing:
  `route_planner_screen.dart:513-524` currently catches the offline failure, shows an error
  toast, and leaves the user stuck on the planner unable to finish a route at all.

### Claude's Discretion

- Where the Dart conversion module lives and whether it mirrors the TS file layout.
- The corpus's on-disk directory layout and expected-values file format.
- How PORT-03 ("`/trail/convert` called for none of them") is proven — test, grep assertion,
  or both.
- What the route planner writes into `duration` (Valhalla estimate vs GPX-derived).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The contract being ported (Phase 33 output)
- `web/src/lib/models/gpx/gpx-metrics-computation.ts` — the algorithm to port. Note the
  `finalElevationGain`/`finalElevationLoss` getters vs the monotonic `total*Smoothed` fields;
  `parseElevation`'s semantics (genuine `0` is data, missing/empty/non-numeric is `undefined`);
  thresholds `(5, 5)`; the defer-then-publish noise filter.
- `web/src/lib/models/gpx/gpx.ts` — `getTotals()`, the assembly of the public metrics.
- `web/src/lib/util/gpx_util.ts` — `gpx2trail()` (the port target) and `fromFile()` (the
  transcoding that stays server-side).
- `.planning/phases/33-conversion-correctness/33-CONTEXT.md` — Phase 33's locked decisions.
  **D-05 is amended by D-01 above**; D-01/D-02 explain why `cumulativeDistance` exists.
- `.planning/phases/33-conversion-correctness/33-VERIFICATION.md` — what "correct" means,
  and the CONV-01..05 defect cases the corpus must cover.

### The endpoint being changed
- `web/src/routes/api/v1/trail/convert/+server.ts` — current behaviour incl. the
  reverse-geocode step (`:87-99`) that moves to the app per D-07, and the OpenAPI JSDoc
  block that must be regenerated.

### The app side
- `app/lib/util/gpx_util.dart` — existing `sanitizeGpxEmail` (always route
  `GpxReader().fromString()` through it) and `buildGpxFromPoints`; `gpx: ^2.3.0`.
- `app/lib/util/trail_import_util.dart` — `importTrailFile`, the live `/trail/convert`
  caller (`:83`).
- `app/lib/routes/route_planner_screen.dart` — `_onFinish` / `finishPlanning`, the
  planner's `/valhalla/height` + `/trail/convert` round trip and its offline dead end
  (`:513-524`).
- `app/lib/routes/navigation_screen.dart` — the recording save path; already calls the
  options sheet at `:733`.
- `app/lib/provider/navigation_stats_provider.dart` — `NavigationStats.elapsed` is already
  moving time; see the tick at `:229-235` and `_pausedAccum`.
- `app/lib/components/navigation/track_save_options_sheet.dart` — the sheet to reuse.

### The web write path that motivates D-10
- `web/src/routes/trail/edit/[id]/+page.svelte` — `updateTotals()` (single call site, inside
  `updateTrailWithRouteData()`), and the ~14 route-mutation paths that reach it.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `showTrackSaveOptionsSheet` — already returns exactly the two toggles needed
  (`recalcHeights`, `followRoads`), both off by default; already wired into the recording
  path. Extending it to planner + import is reuse, not new UI.
- `NavigationStats` — already computes moving time, distance, and elevation gain/loss live,
  and `ActiveNavigationEntity.pausedAccumSeconds` already persists the accumulator.
- `sanitizeGpxEmail` — mandatory pre-parse step for the `gpx` package.
- `app/test/` already has `util/`, `models/`, `provider/` — a home for the corpus test exists.

### Established Patterns
- Phase 33's Vitest precedent is inline fixtures (D-05); the on-disk corpus is a deliberate,
  documented exception (D-01), not a break with convention.
- The app's error-on-offline pattern (`importTrailFile`, `_onFinish`) is toast-and-stay.
  D-15/D-16 replace it with skip-and-proceed for these paths.

### Integration Points
- `trail_import_util.dart:83` — the `/trail/convert` POST becomes transcode-only; the app
  then computes locally and shows the options sheet when online.
- `route_planner_screen.dart` `_onFinish` — drops the unconditional height correction.
- A PocketBase migration is required for `moving_duration` (`db/migrations/`).

</code_context>

<specifics>
## Specific Ideas

- "We already display the time in movement to the user. For a recording/navigation session
  this value is the trail's duration." — the source of D-09; the port must not invent a
  second moving-time derivation.
- "Show these options only when online. When offline skip directly to trail_create_screen."
  — D-15.
- "importTrailFile should show the same sheet with snapToRoad and recalculate heights
  options." — D-15.

</specifics>

<deferred>
## Deferred Ideas

- **Auto-pause tuning / threshold-based moving time** — not needed; the existing stationary
  detection already feeds `_pausedAccum` (D-09). Revisit only if the existing detection
  proves inadequate in the field.
- **Recomputing already-stored trail metrics** (CONV-F01) — still deferred from Phase 33.
  Existing rows keep their pre-Phase-33 values.
- **Displaying `moving_duration` in the web UI** — the field and the display rule are in
  scope; any richer web presentation (e.g. showing both elapsed and moving) is not.

### Reviewed Todos (not folded)
- `2026-07-31-trail-create-screen-offline-gaps.md` (matched 0.9) — **belongs to Phase 35**
  per the ROADMAP's sequencing rationale, which names this todo explicitly as Phase 35 work.
  Not folded despite the high keyword score.
- `2026-07-18-way-types-and-surfaces-breakdown.md` (0.6) — unrelated feature, no dependency
  on the port.
- `2026-07-24-comaps-poly-region-extraction-spike.md` (0.2) — unrelated, backend/regions.

</deferred>

---

*Phase: 34-Dart Conversion Port*
*Context gathered: 2026-07-31*
