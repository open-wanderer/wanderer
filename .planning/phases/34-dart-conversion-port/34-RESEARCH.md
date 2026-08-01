# Phase 34: Dart Conversion Port - Research

**Researched:** 2026-07-31
**Domain:** Cross-language (TypeScript ↔ Dart) port of GPX→trail metrics computation; Flutter/PocketBase/SvelteKit integration
**Confidence:** HIGH (the two highest-risk questions — GPX parser fidelity and float agreement — were empirically verified on this machine, not assumed)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Shared fixture corpus (PORT-02)**
- **D-01:** The corpus lives **on disk in a language-neutral format** (GPX input + expected-metrics file per fixture), read by both the Vitest suite and the Dart tests. This amends Phase 33's D-05 ("fixtures are inline, not disk files") to: *inline for TS-only unit tests, on-disk for the cross-language contract*.
- **D-02:** Expected values are **hand-derived from first principles for the CONV-01..05 defect cases**, with the derivation documented alongside each fixture. Bulk realistic-track fixtures may be **seeded from the corrected TS output and human-reviewed** before locking. The TS implementation is explicitly **not** the oracle.
- **D-03:** The corpus asserts a **tight absolute tolerance, documented per field**: distance and elevation to ~1e-6 m; duration, point counts, and bounding box exact. `dart:math` and V8 trig are not required to agree bit-for-bit.
- **D-04:** The corpus asserts **public metrics only** — distance, elevationGain/Loss, duration, boundingBox, centroid, waypoints, name, description. Dart internals may differ. **`cumulativeDistance` is NOT ported** (its only consumer, the web crop slider, has no app equivalent). The port must reproduce the values `finalElevationGain`/`finalElevationLoss` produce, **not** `totalElevationGainSmoothed`/`LossSmoothed`. Porting the wrong one of the pair is the single most likely way to fail the corpus.

**Convert endpoint (PORT-04)**
- **D-05:** **Hard break** — change the response in place, no `/v2` and no content negotiation.
- **D-06:** Success response is the **raw GPX document** with `Content-Type: application/gpx+xml`. Errors keep the existing JSON shape via `handleError`. The multipart and JSON/raw-text *input* branches stay as they are.
- **D-07:** The reverse-geocode step the endpoint performs today (`:87-99`) **moves to the app** as a separate, optional call made only when online. Offline, `location` is simply empty.
- **D-08:** The **web frontend is unchanged**. It already computes client-side via `gpx2trail` and only uses the endpoint for kml/kmz/tcx/fit transcoding.

**Moving time (CONV-06)**
- **D-09:** **No new derivation is needed.** `NavigationStats.elapsed` is already moving time.
- **D-10:** Moving time is stored in a **new, separate `moving_duration` field**; `duration` keeps exactly one meaning everywhere — GPX-derived elapsed. Display rule: show `moving_duration` when present, else `duration`. (Load-bearing decision — see CONTEXT.md's full rationale on `updateTotals()`'s ~14 web call sites.)
- **D-11:** A recording takes **only `moving_duration`** from the session. Distance, elevation gain/loss, and `duration` all come from the ported Dart computation over the recorded GPX.
- **D-12:** CONV-06 **cannot be pinned by the shared corpus** (moving time is not a function of a GPX). It needs its own Dart-side test on the session → trail hand-off.
- **D-13:** The Dart conversion accepts an **optional duration override** so one conversion path serves all three sources: GPX-derived elapsed when absent, session value when passed.

**Elevation correction / online-only options**
- **D-14:** The Dart conversion is **pure and offline by construction** — GPX in, metrics out, no network. Elevation correction and road-snapping are **caller-driven steps outside it**.
- **D-15:** `showTrackSaveOptionsSheet` becomes the single post-capture gate for **all three** sources, shown **only when online**; offline the app **skips it and goes straight to `trail_create_screen`**.
- **D-16:** This fixes a real offline defect in passing: `route_planner_screen.dart:513-524` currently strands the offline user.

### Claude's Discretion
- Where the Dart conversion module lives and whether it mirrors the TS file layout.
- The corpus's on-disk directory layout and expected-values file format.
- How PORT-03 ("`/trail/convert` called for none of them") is proven — test, grep assertion, or both.
- What the route planner writes into `duration` (Valhalla estimate vs GPX-derived).

### Deferred Ideas (OUT OF SCOPE)
- Auto-pause tuning / threshold-based moving time.
- Recomputing already-stored trail metrics (CONV-F01) — deferred from Phase 33.
- Displaying `moving_duration` in the web UI beyond the D-10 display rule.
- The two `trail_create_screen` offline gaps (blank map, throwing tag autocomplete) — Phase 35.
- Any change to the Phase 33 TS computation itself beyond what the corpus reader needs.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PORT-01 | App computes name/description/waypoints/start/date/distance/elevation/duration/bbox from GPX on-device, no network | "The GPX Parsing Landmine" + "The Elevation Algorithm to Port" + "Code Examples" sections give the exact algorithm and the parser-fidelity fix needed to implement this correctly |
| PORT-02 | Shared fixture proves Dart/TS parity, covering CONV-01..05 | "Fixture Corpus Location" section (empirically verified Dart file-read behavior) + "Float Agreement" section (empirically verified tolerance) |
| PORT-03 | Recordings, planner, `.gpx` import all use the Dart path; `/trail/convert` called for none | "The Three Capture Paths" section maps the single existing call site (`convertGpxToTrail`) and its three callers |
| PORT-04 | `/trail/convert` transcodes only, returns GPX; OpenAPI updated | "The Convert Endpoint" section, including the exact JSDoc occurrences to update |
| PORT-05 | kml/kmz/tcx/fit import online still produces a correct trail via app-side measurement | "The Three Capture Paths" — `importTrailFile` already re-parses `expand.gpxData` client-side; the port only changes what computes the metrics, not the transcode step |
| CONV-06 | Recorded trail reports moving time, imported file reports elapsed | "Moving Time" section confirms `NavigationStats.elapsed`/`pausedAccum` are already correct and traces the `moving_duration` schema/model changes needed |
</phase_requirements>

## Summary

This phase has one central technical risk and it is empirically confirmed, not hypothetical:
**`package:gpx` (`GpxReader`) throws uncaught exceptions on inputs the corrected TS parser
handles gracefully.** An empty-but-present `<ele></ele>` tag — the exact fixture Phase 33's own
test suite uses for its CONV-03 regression guard (`gpx-metrics-computation.test.ts:42`) — crashes
`GpxReader().fromString()` with `FormatException: Invalid double`. So do a non-numeric `<ele>`, a
whitespace-only `<ele>`, an empty `<time></time>`, and a `<trkpt>` missing its `lat`/`lon`
attribute (`StateError: Bad state: No element`). All five were reproduced directly against the
installed `gpx: ^2.3.0` package on this machine. A naive line-for-line port that feeds the shared
corpus's GPX fixtures straight into `GpxReader` will crash on fixture #1, not silently disagree —
which is actually the best possible failure mode (loud, not silent), but it means the port
**cannot** be "parse with `GpxReader`, then replicate the TS algorithm" as a single step. It needs
a pre-parse sanitization pass, following the exact precedent the codebase already established for
`<email>` tags (`sanitizeGpxEmail` in `gpx_util.dart`).

The second-highest risk — whether `dart:math` and V8 agree closely enough for the corpus's 1e-6 m
tolerance — turned out to be a non-issue in practice: a 5000-point haversine accumulation (every
`sin`/`cos`/`atan2`/`sqrt` call in the hot path) run independently in Node/V8 and the Dart VM
produced **bit-identical output to 17 significant digits** (`36221.778933403148` in both). D-03's
1e-6 m tolerance is not just realistic, it is generous by many orders of magnitude — on this
platform pair, algorithmic agreement is the only thing that can fail, not floating-point drift.

The third question — whether Dart's plain `test`/`flutter test` runner can read fixtures from
disk — is also resolved empirically: `flutter test` sets the process working directory to the
package root (`app/`), and plain `dart:io` `File` reads with relative paths work with no asset
bundle involved. The asset-bundle restriction that governs `rootBundle.load()` inside a running
app does not apply to `flutter test`/`dart test`. This means the on-disk corpus can live at the
repo root (sibling to `web/`, `app/`, `db/`) and be read via a relative path from each toolchain's
own root — no `pubspec.yaml` `assets:` entry, no symlink, no copy step needed.

**Primary recommendation:** Port `gpx-metrics-computation.ts` and `gpx.ts`'s `getTotals()` as a
new Dart module (new file, not an extension on the existing `Gpx` class in `gpx_util.dart` —
that file already has a *different*, CONV-01-buggy `getTotals()` extension consumed by
`elevation_profile.dart`/`trail_panel.dart`; colliding names would be actively dangerous). Add a
`sanitizeGpxEleAndTime` (or similarly named) pre-parse pass alongside the existing
`sanitizeGpxEmail`, applied at every `GpxReader().fromString()` call site that feeds
locally-captured or imported data. Reuse the existing `SphericalGreatCircle.distanceTo(point,
radius: 6371000)` (from `package:geobase`, re-exported via `package:maplibre`, already used in
`navigation_stats_provider.dart`/`gpx_util.dart`) for the haversine step — it is formula-identical
to `web/src/lib/models/gpx/utils.ts`'s `haversineDistance` (same radius, same formula, only the
degree→radian conversion order differs, which is float-noise-level and irrelevant given the 1e-6 m
tolerance).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| GPX parsing (XML → in-memory model) | Client (app) | — | `package:gpx`'s `GpxReader`, entirely on-device, no network (PORT-01) |
| GPX→trail metrics computation (distance/elevation/duration/bbox/centroid) | Client (app) | — | The whole point of the phase: ported from `gpx-metrics-computation.ts`/`gpx.ts`, pure, offline (D-14) |
| kml/kmz/tcx/fit → GPX transcoding | API / Backend | — | Explicitly kept server-side (out of scope; vendored `toGeoJSON`/JSZip/`fit-parser` deps, PORT-05) |
| Reverse geocoding (place name from start point) | Client (app) | API / Backend (Nominatim proxy) | Moves from the convert endpoint to an app-side, online-only, optional call (D-07) |
| Elevation correction (Valhalla height) | Client (app), orchestration | API / Backend (Valhalla proxy) | Caller-driven step outside the pure conversion (D-14); app decides *when* via `showTrackSaveOptionsSheet` (D-15) |
| Road-snapping (Valhalla trace_route) | Client (app), orchestration | API / Backend (Valhalla proxy) | Same as above; both gated by the same online-only sheet |
| Moving-time derivation | Client (app) | — | Already computed live by `NavigationStatsNotifier`; no new derivation, no server involvement (D-09) |
| Trail persistence / schema (`moving_duration`) | Database (PocketBase) | Client (app), Frontend Server (web edit page) | New nullable field must propagate through migration → TS model → Dart model/entity → OpenAPI schema |
| Shared fixture corpus | — (build/test-time artifact) | — | Language-neutral, read by both Vitest (Frontend Server tier tests) and `flutter test` (Client tier tests); not a runtime capability |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `gpx` | ^2.3.0 (already a direct dependency, `app/pubspec.yaml:36`) | GPX 1.1 parse/write | Already the app's only GPX library (`gpx_util.dart`, `trail_import_util.dart`, `route_planner_handoff_util.dart` all depend on it); no alternative needed or in scope |
| `maplibre` (re-exporting `geobase`'s `SphericalGreatCircle`) | already installed (`0.3.5` pinned per STATE.md 18-02) | Haversine distance | Already used for the identical calculation in `navigation_stats_provider.dart` and the existing (buggy) `getTotals()` in `gpx_util.dart` — formula-verified identical to the TS `haversineDistance` (see "Code Examples") |

No new external packages are required by this phase. `gpx: ^2.3.0` and `maplibre` are already
direct dependencies; the port is new Dart *code*, not a new dependency.

**Version verification (VERIFIED via local pub cache, this machine):**
```
$ grep "gpx:" app/pubspec.yaml
  gpx: ^2.3.0
$ ls ~/.pub-cache/hosted/pub.dev/ | grep gpx
  gpx-2.3.0
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `package:gpx`'s `GpxReader` directly | A hand-rolled XML-event parser mirroring `gpx.ts`'s exact permissive semantics (undefined-on-anything-malformed) | Would sidestep the crash-on-malformed-tag landmine entirely, but throws away a maintained, already-integrated dependency for a large amount of new hand-rolled XML code — not justified when a pre-parse sanitize pass (following the `sanitizeGpxEmail` precedent) fixes the same problem in ~10 lines |
| Reusing `SphericalGreatCircle.distanceTo` | Porting `utils.ts`'s `haversineDistance` verbatim as a private function in the new module | Both are numerically equivalent (verified: same formula, same 6371000 m radius). Reuse is DRY and matches the codebase's existing convention; verbatim port is marginally more obviously "this is the ported function," at the cost of duplicating a formula that already exists three times in the app. Recommend reuse; the fixture corpus will catch it immediately if wrong. |

## Package Legitimacy Audit

Not applicable — this phase introduces no new external packages. `gpx` and the `maplibre` package
(via its `geobase` dependency for `SphericalGreatCircle`) are already installed, direct
dependencies, exercised in production code paths today.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         THREE CAPTURE PATHS                          │
│                                                                        │
│  Recording          Route Planner        .gpx/.kml/.kmz/.tcx/.fit    │
│  (navigation_       (route_planner_      import                     │
│   screen.dart)       screen.dart /       (trail_import_util.dart)    │
│       │              route_planner_             │                    │
│       │              handoff_util.dart)          │                    │
│       │                   │                      │                    │
│       ▼                   ▼                      ▼                    │
│  breadcrumb Wpts     buildFinalPlannedGpx   file bytes                │
│  + NavigationStats   (legs → Gpx)                │                    │
│  .elapsed (moving)        │                      │                    │
│       │                   │                 ┌────┴────┐               │
│       │                   │                 │ .gpx?   │──No──┐        │
│       │                   │                 └────┬────┘      │        │
│       │                   │                     Yes│          ▼        │
│       │                   │                       │    POST /trail/   │
│       │                   │                       │    convert        │
│       │                   │                       │    (SERVER,       │
│       │                   │                       │    transcode-only,│
│       │                   │                       │    PORT-04)       │
│       │                   │                       │         │        │
│       │                   │                       │◄────────┘        │
│       │                   │                       │  raw GPX XML     │
│       ▼                   ▼                       ▼                   │
│  ┌───────────────────────────────────────────────────────┐          │
│  │        sanitizeGpxEmail + [NEW] sanitizeGpxEle...      │          │
│  │        GpxReader().fromString(sanitized)               │          │
│  └───────────────────────────────────────────────────────┘          │
│                          │                                            │
│                          ▼                                            │
│  ┌───────────────────────────────────────────────────────┐          │
│  │   [NEW] Dart port of gpx-metrics-computation.ts        │          │
│  │   + gpx.ts's getTotals() (pure, offline, D-14)          │          │
│  │   in: Gpx, optional movingDurationOverride (D-13)       │          │
│  │   out: name/desc/waypoints/start/date/distance/         │          │
│  │        elevationGain/Loss/duration/movingDuration/bbox  │          │
│  └───────────────────────────────────────────────────────┘          │
│                          │                                            │
│                          ▼                                            │
│              ┌───────────────────────┐                                │
│              │  Online? (D-15)        │                                │
│              └──────┬─────────┬──────┘                                │
│                  Yes│          │No                                    │
│                     ▼          ▼                                      │
│         showTrackSaveOptionsSheet   trail_create_screen                │
│         (recalcHeights/followRoads) (skip sheet, D-15/D-16)           │
│                     │                                                 │
│         Valhalla /height, /trace-route (D-14: caller-driven,          │
│         re-runs the SAME pure conversion after writing new GPX)       │
│                     │                                                 │
│                     ▼                                                 │
│              trail_create_screen                                      │
└─────────────────────────────────────────────────────────────────────┘

              ┌──────────────────────────────────────────┐
              │  fixtures/gpx-corpus/ (repo root, NEW)     │
              │  read by:                                  │
              │   - web/src/lib/models/gpx/*.test.ts       │
              │     (Vitest, relative path from web/)      │
              │   - app/test/util/*_test.dart              │
              │     (flutter test, relative path from      │
              │      app/ — VERIFIED CWD = package root)   │
              └──────────────────────────────────────────┘
```

### Recommended Project Structure
```
app/lib/
├── util/
│   ├── gpx_util.dart              # existing: sanitizeGpxEmail, buildNavShape,
│   │                               #   buildGpxFromPoints, the OLD GpxMappingUtils.getTotals()
│   │                               #   extension (do not touch/collide — see Pitfall 5)
│   └── gpx_conversion_util.dart   # NEW (suggested name, Claude's discretion per CONTEXT.md):
│                                   #   sanitizeGpxEleAndTime + the ported
│                                   #   GpxMetricsComputation class + computeTrailMetrics()
web/src/lib/models/gpx/
├── gpx-metrics-computation.ts     # PORT SOURCE — do not modify beyond what the
│                                   #   corpus reader needs (per CONTEXT.md scope)
└── gpx.ts                         # PORT SOURCE (getTotals()) — same constraint
fixtures/                          # NEW, repo root — sibling to app/, web/, db/
└── gpx-corpus/
    ├── 01-two-point-segment/
    │   ├── input.gpx
    │   └── expected.json          # format: Claude's discretion (D-01 constrains
    │                               #   location, not shape)
    ├── 02-partial-elevation/
    ├── 03-missing-vs-empty-ele/   # THE landmine fixture — <ele></ele> present
    ├── 04-switchback-scramble/
    ├── 05-jittery-track/
    └── 06-multi-anchor-planned-route/
```

### Pattern 1: Pre-parse sanitization before `GpxReader` (mandatory, new)

**What:** A defensive string-rewrite pass, run before every `GpxReader().fromString()` call on
locally-captured or imported GPX, that neutralizes the five confirmed crash inputs.

**When to use:** Every call site that currently calls `GpxReader().fromString(sanitizeGpxEmail(...))`
on data that did not just come from this app's own `GpxWriter().asString(...)` (i.e., anything
that could be malformed: imported files, or any future third-party GPX source). Locally-generated
GPX (`buildGpxFromPoints`, `mergeHeightsIntoGpx`, `buildFinalPlannedGpx`) is safe by construction
and does not strictly need it, but applying it unconditionally is cheap insurance and keeps one
code path.

**Example (VERIFIED — the exact five failure modes, reproduced on this machine against the
installed `gpx: ^2.3.0`):**
```dart
// Source: empirically reproduced via `dart run --packages=.dart_tool/package_config.json`
// against the real installed gpx-2.3.0 package on 2026-07-31.
//
//   missing ele entirely:              OK   ele=null   (matches TS undefined)
//   self-closed <ele/>:                OK   ele=null   (matches TS undefined)
//   open/close <ele></ele>:            THREW FormatException: Invalid double
//   whitespace-only <ele> </ele>:      THREW FormatException: Invalid double
//   non-numeric <ele>N/A</ele>:        THREW FormatException: Invalid double
//   genuine <ele>0</ele>:              OK   ele=0.0    (matches TS real zero)
//   missing lat attribute on <trkpt>:  THREW StateError: Bad state: No element
//   missing lon attribute on <trkpt>:  THREW StateError: Bad state: No element
//   empty <time></time>:               THREW FormatException: Invalid date format
//   pretty-printed <ele>\n 1000.5\n</ele>: OK ele=1000.5 (double.parse trims whitespace
//                                          around an otherwise-valid number — only a
//                                          tag with NO valid number left after trim throws)
```
The fix strips empty/whitespace/non-numeric `<ele>...</ele>` and `<time>...</time>` element
bodies (turning them into self-closing `<ele/>`/`<time/>`, which `GpxReader` already handles as
`null`), mirroring `sanitizeGpxEmail`'s existing regex-rewrite approach — no new parsing
dependency, no fork of `GpxReader`. A `<trkpt>` missing `lat`/`lon` is a much rarer, more
structurally broken input (the GPX spec requires both); flag it as a lower-priority pitfall (Common
Pitfalls) rather than building full recovery for it, since Phase 33's own CONV-01..05 fixtures
never test a missing coordinate attribute — but a real-world malformed import file could still hit
it, so wrapping the whole `importTrailFile` parse in a try/catch with the existing error-toast
path (already there) is sufficient defense.

### Pattern 2: The elevation defer-then-publish algorithm (port verbatim, do not simplify)

**What:** `GpxMetricsComputation.addAndFilter()` (`gpx-metrics-computation.ts:97-235`) holds each
above-threshold elevation excursion in a single `pendingDelta` slot rather than committing it
immediately, and only discards it (as noise) when the track returns to the pre-excursion elevation
*without* having moved horizontally. This is precisely what makes `totalElevationGainSmoothed`/
`LossSmoothed` monotonic (required by `trail_anchor_list.svelte`'s per-segment differencing on the
web side, though that consumer itself is out of scope) while still crediting genuine
low-horizontal-movement climbs (CONV-04) and excluding stationary GPS/altimeter noise round-trips.

**When to use:** This is the entire body of the ported `computeTrailMetrics`/equivalent. Do not
attempt to simplify to a plain gain/loss accumulator — that reintroduces exactly the CONV-04 bug
Phase 33 fixed.

**Critical distinction (D-04, restated because it is the single most likely way to fail the
corpus):** A *completed* track's reported gain/loss must come from `finalElevationGain`/
`finalElevationLoss` — which include the still-pending, unconfirmed excursion — **not**
`totalElevationGainSmoothed`/`totalElevationLossSmoothed`, which deliberately exclude it to stay
monotonic for a per-segment-differencing consumer the Dart side does not have. The port only ever
needs the `final*` semantics; there is no Dart-side consumer that needs the monotonic smoothed
fields at all, so the ported class can be simpler than the TS original — it can compute
`finalElevationGain`/`finalElevationLoss` directly as its only public elevation output, while
still implementing the internal defer/publish/discard state machine to get the *value* right.

```typescript
// Source: web/src/lib/models/gpx/gpx-metrics-computation.ts:74-95 (verified against
// 33-VERIFICATION.md's confirmed-passing shipped code)
get finalElevationGain(): number {
  return this.totalElevationGainSmoothed + Math.max(this.pendingDelta, 0);
}
get finalElevationLoss(): number {
  return this.totalElevationLossSmoothed + Math.max(-this.pendingDelta, 0);
}
private publishPending() {
  if (this.pendingDelta > 0) this.totalElevationGainSmoothed += this.pendingDelta;
  else if (this.pendingDelta < 0) this.totalElevationLossSmoothed -= this.pendingDelta;
  this.pendingDelta = 0;
  this.pendingAnchorZ = null;
}
```

### Pattern 3: Distance — smoothed, not raw; `cumulativeDistance` not ported

**What:** `gpx.ts:148`'s `totalDistance = metrics.totalDistanceSmoothed` — the reported distance
is the threshold-gated accumulator, not the raw per-point haversine sum. Per D-04, the raw,
index-aligned `cumulativeDistance` array (`gpx-metrics-computation.ts:56,114,138`) exists in TS
solely to feed the web trail-edit crop slider, which has no app equivalent, and must **not** be
ported — it would be dead code with no consumer.

### Pattern 4: Bounding box / centroid — include index 0

**What:** CONV-01/CONV-02's fix: the loop over track points must start at `i = 0` (not `i = 1`),
and the centroid must divide by the same count it summed (`summedPointCount`, not
`allPoints.length`). This is a one-line-class bug in the TS original that the Dart port must not
reintroduce independently — it is trivial to get right the first time in Dart since there is no
pre-existing buggy version to accidentally copy from (the *only* existing Dart `getTotals()`
extension in `gpx_util.dart`, unrelated to this port, has the identical `for (int i = 1; ...)`
bug — see Pitfall 5 below; do not use it as a reference implementation).

### Anti-Patterns to Avoid
- **Reusing/overloading the existing `GpxMappingUtils.getTotals()` extension in `gpx_util.dart`:**
  It is a different, older, CONV-01-style-buggy implementation (`for (int i = 1; ...)`, no
  defer-then-publish noise filter, raw distance not smoothed) already consumed by
  `elevation_profile.dart` and `trail_panel.dart` for **unsaved-GPX preview display**. Silently
  changing its behavior as a side effect of this phase, or naming the new port function the same
  thing, risks either breaking those two display call sites or producing two functions with the
  same name and different semantics in the same file. See Common Pitfalls below for the scope
  decision this raises.
- **Feeding untrusted GPX straight into `GpxReader().fromString()`:** Confirmed to throw
  uncaught exceptions on inputs the TS parser handles gracefully. Every new parse call site on
  potentially-malformed data needs the sanitize pass (Pattern 1) or a wrapping try/catch with the
  existing error-toast fallback.
- **Hand-deriving a new haversine formula instead of reusing `SphericalGreatCircle`:** Verified
  formula-identical (same 6371000 m radius, same haversine terms) to the TS original; introducing
  a second, subtly different implementation risks a real (not float-noise) divergence.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Great-circle distance between two lat/lon points | A new haversine implementation | `SphericalGreatCircle(point).distanceTo(other, radius: 6371000)` from `package:geobase` (via `maplibre`), already imported and used identically in `navigation_stats_provider.dart` and the existing `gpx_util.dart` `getTotals()` | Verified formula-identical to `utils.ts`'s `haversineDistance`; a second hand-rolled formula is a real (non-float-noise) divergence risk with zero benefit |
| GPX XML parsing | A custom XML-event GPX reader | `package:gpx`'s `GpxReader`, with a pre-parse sanitize pass for the five confirmed crash inputs | Already the app's integrated, maintained GPX library; the sanitize pass is ~10 lines vs. reimplementing a GPX parser |
| Malformed-XML defensive parsing | Wrapping every field read in try/catch inside the metrics loop | A single upfront string-sanitize pass (Pattern 1), mirroring `sanitizeGpxEmail`'s existing precedent | One place to reason about malformed input, not scattered try/catch; matches the codebase's established convention for this exact class of problem |

**Key insight:** This phase's "don't hand-roll" risk is not about avoiding a third-party library —
it is about avoiding a *second, silently different* implementation of an algorithm that already
exists twice in this codebase (the TS original being ported, and the older buggy Dart
`getTotals()` extension it must not be confused with).

## Common Pitfalls

### Pitfall 1: `GpxReader` throws on inputs the TS parser treats as "no data" (VERIFIED)
**What goes wrong:** `GpxReader().fromString()` throws `FormatException`/`StateError` on an
empty-but-present `<ele></ele>`, a whitespace-only or non-numeric `<ele>`, an empty `<time></time>`,
or a `<trkpt>` missing `lat`/`lon` — all of which the TS parser (`isomorphic-xml2js` +
`parseElevation`) handles by returning `undefined`.
**Why it happens:** `package:gpx`'s reader calls `double.parse`/`DateTime.parse` directly on the
accumulated element text with no empty/malformed guard, and reads `lat`/`lon` via
`.firstWhere(...)` with no `orElse`.
**How to avoid:** Pre-parse sanitize pass (Pattern 1) before every `GpxReader().fromString()` call
on non-self-generated GPX.
**Warning signs:** The shared fixture corpus's CONV-03 fixture (which Phase 33's own TS suite
already tests with a literal `<ele></ele>`) is the canary — if it's added to the on-disk corpus
without the sanitize fix, the Dart test crashes on load, not on assertion.

### Pitfall 2: Confusing `finalElevationGain`/`Loss` with `totalElevationGainSmoothed`/`LossSmoothed`
**What goes wrong:** Porting the monotonic smoothed fields as the public output instead of the
`final*` getters silently under-reports a track that ends mid-climb (the pending excursion is
real but unconfirmed, and only the `final*` getters surface it).
**Why it happens:** The names are easy to conflate, and the TS class exposes both because a *web*
consumer (`trail_anchor_list.svelte`, out of scope) needs the monotonic ones for per-segment
differencing — a distinction that doesn't obviously matter until you know that consumer exists.
**How to avoid:** The Dart port has no per-segment-differencing consumer; expose only `final*`
semantics as the port's public elevation output (see Pattern 2).
**Warning signs:** A corpus fixture that "ends mid-swing" (the exact CR-03/33-04 regression class
from Phase 33's verification history) reports lower gain/loss than expected.

### Pitfall 3: Porting `cumulativeDistance` anyway
**What goes wrong:** Wasted Dart code with no consumer (D-04 explicitly rules this out), plus it
resurrects the exact "index-aligned but nobody checks alignment" class of bug D-01/D-02 in Phase
33 had to repair.
**How to avoid:** The ported metrics class's public surface should be exactly: distance
(smoothed), elevationGain/Loss (`final*` semantics), duration, boundingBox, centroid, waypoints,
name, description — matching D-04 verbatim.

### Pitfall 4: The convert endpoint's OpenAPI schema is defined in 6 separate JSDoc blocks
**What goes wrong:** Updating only the convert route's own JSDoc block (`+server.ts:8-39`)
leaves the `Trail` schema (which also needs the new `moving_duration` field for CONV-06) stale in
the other 5 locations.
**Why it happens:** `web/src/lib/models/api/openapi_schemas.ts` defines the Trail-shaped schema
(with `elevation_gain`/`elevation_loss`/`duration` triples) independently at **6 separate JSDoc
locations** (verified via grep: lines ~390, 467, 522, 627, 668, 692) rather than a single shared
`$ref`.
**How to avoid:** Grep for `elevation_gain` in `openapi_schemas.ts` before editing, update every
occurrence, then run `npm run openapi:generate` (`web/scripts/generate-openapi.js`, which reads
`openapi.config.js`'s `baseSchemasPath`) and diff the regenerated
`static/docs/api/wanderer.openapi.json`.

### Pitfall 5: Two `getTotals()`-shaped functions on `Gpx` with different correctness
**What goes wrong:** `app/lib/util/gpx_util.dart`'s existing `GpxMappingUtils.getTotals()`
extension (used by `elevation_profile.dart:178` and `trail_panel.dart:44`, both **unsaved-GPX
preview** call sites — i.e., exactly the route-planner/import preview scenario before a trail is
saved) has the *same* CONV-01-shaped bug as the pre-Phase-33 TS code (`for (int i = 1; ...)`
drops the first point/hop) and no defer-then-publish elevation filter at all. It is a live,
separate implementation from what this phase ports.
**Why it happens:** It was written independently of the TS algorithm, for a narrower
display-only purpose (a chart's max/min stats), before Phase 33's fixes existed to reference.
**How to avoid:** Name the new ported function distinctly (e.g. `computeTrailMetrics`, not
`getTotals`) so the two coexist without collision. **Scope decision for the planner:** whether to
also fix/redirect `elevation_profile.dart`/`trail_panel.dart` onto the new correct implementation
is a judgment call — CONTEXT.md's phase boundary doesn't mention these two files, but leaving them
on the old buggy path means a previewed-but-unsaved trail's elevation-profile chart will show
different numbers than the trail's persisted (Dart-ported) stats once saved. Flagging this
explicitly rather than silently leaving it, mirroring how Phase 33 flagged (but didn't fix)
`trail_anchor_list.svelte`'s analogous out-of-scope drift.

### Pitfall 6: Waypoint icon symbol mapping — TS uses Font Awesome name strings, Dart's model expects `FaIconData`
**What goes wrong:** `gpx2trail` sets `wp.icon = wpt.sym` only `if (wpt.sym && icons.includes(wpt.sym))` where `icons` (`web/src/lib/util/icon_util.ts:3-1393`) is a ~1400-entry array of kebab-case Font Awesome icon names. The Dart `Waypoint.icon` field is typed `FaIconData` (`app/lib/models/waypoint.dart:20`), deserialized from a string via `FaIconDataConverter.fromJson` → `fontAwesomeIconsMap[json] ?? FontAwesomeIcons.circle` (`app/lib/models/converter/fa_icon_data_converter.dart`). A locally-constructed Dart port that builds `Waypoint` objects directly (bypassing server JSON) must replicate this same lookup — `fontAwesomeIconsMap[wpt.sym] ?? FontAwesomeIcons.circle` — not assign the raw string.
**How to avoid:** Route the ported waypoint-building step through `fontAwesomeIconsMap` (already present in `app/lib/util/icon_util.dart:1018`), matching the existing `FaIconDataConverter` semantics exactly.

### Pitfall 7: `moving_duration` migration must target the live field ID/collection ID
**What goes wrong:** Guessing at PocketBase collection/field identifiers instead of reading them
from the latest schema snapshot risks the migration silently no-op'ing (wrong collection id) or
colliding.
**How to avoid:** Confirmed via `db/migrations/1747064968_collections_snapshot.go` (the latest
full snapshot touching `trails`): the `trails` collection id is `e864strfxo14pm4`; the existing
`duration` field is `{"id": "ukr9rqz4", "type": "number", "min": 0, "max": null, "onlyInt": false}`.
A new `moving_duration` migration should mirror that exact field shape (type `number`, `min: 0`,
`max: null`, `required: false` so it can be genuinely absent, matching D-10's "no value" state)
and use `app.FindCollectionByNameOrId("e864strfxo14pm4")` (or `"trails"`, which also resolves —
both are valid per existing migrations' mixed usage) plus `collection.Fields.AddMarshaledJSONAt(...)`,
following the simple-add-field pattern in `db/migrations/1780000006_trail_external_reference_provider_text.go`
(`collection.Fields.GetByName(...) == nil` guard before adding, for idempotency).

## Code Examples

### The exact haversine formula being ported (VERIFIED formula-identical to `SphericalGreatCircle`)
```typescript
// Source: web/src/lib/models/gpx/utils.ts:23-34
function haversineDistance(lat1, lon1, lat2, lon2): number {
  const R = 6371; // km
  const dLat = (lat2 - lat1) * (Math.PI / 180);
  const dLon = (lon2 - lon1) * (Math.PI / 180);
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1 * (Math.PI / 180)) * Math.cos(lat2 * (Math.PI / 180)) *
    Math.sin(dLon / 2) * Math.sin(dLon / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c * 1000; // metres
}
```
```dart
// Source: ~/.pub-cache/hosted/pub.dev/geobase-1.5.0/lib/src/geodesy/spherical/spherical_great_circle.dart:107-130
// (re-exported by package:maplibre, already imported in app/lib/provider/navigation_stats_provider.dart)
double distanceTo(Geographic destination, {double radius = 6371000.0}) {
  final lat1 = position.lat.toRadians();
  final lon1 = position.lon.toRadians();
  final lat2 = destination.lat.toRadians();
  final lon2 = destination.lon.toRadians();
  final dlat = lat2 - lat1;
  final dlon = lon2 - lon1;
  final a = sin(dlat / 2) * sin(dlat / 2) + cos(lat1) * cos(lat2) * sin(dlon / 2) * sin(dlon / 2);
  final c = 2 * atan2(sqrt(a), sqrt(1 - a));
  return radius * c;
}
```
Same radius (6371 km = 6,371,000 m), same haversine terms, same 2·atan2(√a, √(1−a)) form. Only
the degree→radian conversion order differs (per-point here vs. per-delta in TS) — float-noise
level, irrelevant at the corpus's locked 1e-6 m tolerance (see empirical measurement below).

### Empirical float-agreement measurement (VERIFIED on this machine, 2026-07-31)
A 5000-point haversine accumulation loop, using a pure-double linear congruential PRNG (avoiding
JS/Dart's differing 32-bit bitwise-integer semantics, which would otherwise produce a different
point sequence per language and invalidate the comparison), run independently:
```
Node v22.12.0 (V8):        N=5000 ACCUM_TOTAL 36221.778933403148
Dart SDK 3.12.2 (Dart VM): N=5000 ACCUM_TOTAL 36221.778933403148
```
Bit-identical to all 17 printed significant digits. Individual pairwise `haversineDistance` calls
at 5 spot-check coordinate pairs (including a near-pole case and an antimeridian-crossing case)
matched to at least 14 significant digits, with the one observed difference (3e-12 relative, ~0.3
nm absolute) at the very last representable bit of a `double`. **Confidence: HIGH for this
platform pair (Dart VM JIT / V8), not yet measured against Dart AOT on iOS/Android hardware** —
but `sin`/`cos`/`atan2`/`sqrt` are IEEE-754-conformant, well-behaved functions across
platforms/libm implementations to within a few ULP, so on-device drift, if any, is expected to
remain many orders of magnitude below the locked 1e-6 m tolerance. Recommend keeping the 1e-6 m
tolerance as a defensible safety margin (D-03 is correct as locked) rather than tightening to
exact equality, since exact equality has not been verified on-device.

### `flutter test` file-read behavior (VERIFIED on this machine, 2026-07-31)
```dart
// Verified: a plain flutter_test file reading a fixture via a relative path
// succeeds when the app is invoked via `flutter test` from the package root.
test('reads a fixture file from disk via a relative path', () {
  final file = File('test/_tmp_probe_fixtures/sample.json'); // relative to app/
  expect(file.existsSync(), isTrue); // PASSED
});
// $ cd app && flutter test test/_tmp_probe_disk_read_test.dart
// CWD: /Users/.../wanderer/app
// 00:00 +1: All tests passed!
```
No `pubspec.yaml` `assets:` entry was needed — that restriction governs `rootBundle.load()` calls
made by a *running app* (asset-bundle resolution), not `dart:io` `File` reads inside `flutter
test`/`dart test`, which run with CWD = the package root and full filesystem access. This means a
corpus fixture directory at the repo root (e.g. `fixtures/gpx-corpus/`) is directly readable from
Dart tests via `File('../fixtures/gpx-corpus/...')` and from Vitest tests (whose CWD is `web/`,
per `vite.config.ts`'s lack of a `root` override) via
`fs.readFileSync(path.join(process.cwd(), '../fixtures/gpx-corpus/...'))` or
`path.resolve(__dirname, ...)`. Vitest's `test.include: ['src/**/*.{test,spec}.{js,ts}']` only
scopes which files are *discovered as tests* — it does not sandbox what a discovered test can
read from disk.

### Single Dart call site for `/trail/convert` (VERIFIED via repo-wide grep)
```
$ grep -rn "trail/convert" app/lib/
app/lib/util/trail_import_util.dart:83:      .post('/trail/convert', data: formData);
```
Exactly one call site: `convertGpxToTrail(WidgetRef ref, FormData formData)` in
`trail_import_util.dart:80-108`. All three capture paths funnel through it:
- `importTrailFile` (`trail_import_util.dart:60`) calls it directly.
- `buildDraftTrail` (`route_planner_handoff_util.dart:187-222`) calls it — used by both
  `finishPlanning` (route planner) and `_saveRecordedTrack` (`navigation_screen.dart:845`,
  recording).

This means PORT-03 ("`/trail/convert` called for none of them") reduces to changing the body of
one function (`convertGpxToTrail`) to run the local Dart conversion instead of the HTTP POST, and
its verification reduces to one grep assertion plus the existing call-site tests already exercising
`buildDraftTrail`/`importTrailFile`.

### `moving_duration` propagation surface (VERIFIED via grep across TS/Dart/ObjectBox/OpenAPI)
| Layer | File | Current shape | Change needed |
|-------|------|---------------|----------------|
| PocketBase schema | `db/migrations/` (new file) | `trails` collection, field id `ukr9rqz4` = `duration` (type `number`, `min:0`, `max:null`) | Add sibling field `moving_duration`, same shape, `required: false` |
| TS model | `web/src/lib/models/trail.ts:31` (`duration?: number`) | — | Add `moving_duration?: number` alongside |
| TS display | `web/src/routes/trail/edit/[id]/+page.svelte` | ~14 call sites → `updateTotals()` (single choke point, per D-10's rationale) | Display rule only (D-10): show `moving_duration` when present else `duration`; `updateTotals()` must never write `moving_duration` |
| Dart model | `app/lib/models/trail.dart:76` (`@Default(0) double duration`) + `@JsonKey` fields | — | Add `@JsonKey(name: 'moving_duration') double? movingDuration` |
| Dart ObjectBox entity | `app/lib/entities/trail_entity.dart:26` (`double? duration`) | — | Add `double? movingDuration` field + wire `fromModel`/`toModel` (mirrors existing `duration` handling exactly) |
| OpenAPI schema | `web/src/lib/models/api/openapi_schemas.ts` | `elevation_gain`/`elevation_loss`/`duration` triple appears at **6** separate JSDoc locations (verified via grep) | Add `moving_duration` at all 6, then `npm run openapi:generate` |
| Codegen | `app/pubspec.yaml` dev deps: `build_runner ^2.13.1`, `freezed ^3.2.5`, `json_serializable ^6.13.0`, `objectbox_generator ^5.3.1` | — | `dart run build_runner build --delete-conflicting-outputs` regenerates `trail.freezed.dart`/`trail.g.dart`/ObjectBox model after the field additions (STATE.md's Phase 27 precedent: expect incidental `.g.dart` hash churn in unrelated provider files from the same project-wide pass — not a regression) |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| App POSTs to `/trail/convert` for every capture path, server computes the trail | App parses+computes locally via a ported Dart module; server only transcodes non-GPX formats | This phase (34) | Removes a network dependency for GPX-based capture (recording/planner/`.gpx` import), enabling true offline capture — the prerequisite for Phase 36's local-first recording |
| `gpx.ts`'s pre-Phase-33 buggy metrics (dropped first point, elevation-as-0 on missing `<ele>`, threshold-gated elevation, raw distance) | Corrected TS algorithm (CONV-01..05, verified passing per `33-VERIFICATION.md`) | Phase 33 (immediately prior) | This phase ports the **corrected** algorithm, not the original — porting first would have pinned the bugs into Dart (explicitly why Phase 33 was sequenced before 34) |
| `cumulativeDistance` planned for deletion per the original ROADMAP | Repaired in place (index-aligned, raw) as a **TS-only** field; Phase 34 confirms it is NOT ported to Dart | Phase 33 D-01, reaffirmed by Phase 34 D-04 | The Dart port's public surface is narrower than the TS class's — by design, not omission |

**Deprecated/outdated:**
- The `/trail/convert` endpoint's role as "the trail computer" — after this phase it is
  transcode-only (kml/kmz/tcx/fit → GPX). Any code or documentation describing it as returning
  computed trail metrics is stale after PORT-04 lands.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | On-device (iOS/Android AOT) `dart:math` trig agrees with the desktop Dart VM (JIT) measurement to well within 1e-6 m | "Float Agreement" / Code Examples | If a specific device's libm diverges more than expected, a corpus test could flake on CI (desktop) vs. fail on-device (or vice versa) at the 1e-6 m boundary — low risk given IEEE-754 conformance norms, but not empirically confirmed on mobile hardware in this session |
| A2 | `maplibre` version pinned at `0.3.5` (per STATE.md's Phase 18-02 decision) still re-exports `geobase`'s `SphericalGreatCircle` with the same formula verified here | "Don't Hand-Roll" / Code Examples | If the pinned version's transitive `geobase` differs, the formula-equivalence claim would need re-verification — low risk, `SphericalGreatCircle`'s formula is a stable, unlikely-to-change part of that library |
| A3 | `npm run openapi:generate` is the correct/only regeneration command and needs no other flags for a schema-only field addition | Pitfall 4 | If wrong, the OpenAPI JSON could go stale despite JSDoc edits — low risk, this is a repo `package.json` script, directly verified to exist |
| A4 | Standard `dart run build_runner build --delete-conflicting-outputs` is this repo's codegen convention (not found as an explicit documented command, inferred from dev-dependency presence + STATE.md's Phase 27 anecdote about incidental `.g.dart` churn from "a single project-wide build_runner pass") | Code Examples ("moving_duration propagation surface" table) | Low risk — this is the standard invocation for this exact toolchain (freezed + json_serializable + riverpod_generator + objectbox_generator); if the repo uses a wrapper script instead, the planner should grep for one (none found in Makefile/`.github` in this session) |

**If this table is empty:** N/A — see above; all other claims in this document were empirically
verified on this machine (parser fidelity, float agreement, `flutter test` file access, single
call site, migration field shapes) or directly read from source (algorithm semantics, schema
locations).

## Open Questions (RESOLVED)

1. **(RESOLVED — CONTEXT.md D-17: redirect both consumers and delete the extension; implemented
   by 34-04 Task 3.)** Should `elevation_profile.dart`/`trail_panel.dart` be redirected onto the
   new correct Dart metrics function, or left on the existing buggy `getTotals()` extension?**
   - What we know: Both are unsaved-GPX preview display consumers, not part of the
     save/persist pipeline PORT-01..05 targets. The existing extension has the same class of bug
     Phase 33 fixed in TS (CONV-01-shaped: `for (int i = 1; ...)`).
   - What's unclear: Whether fixing it is in-phase-scope (CONTEXT.md's phase boundary doesn't
     name these files) or a follow-up, mirroring how Phase 33 explicitly deferred
     `trail_anchor_list.svelte`'s analogous drift as out-of-scope debt.
   - Recommendation: Flag explicitly in the plan's notes (as Phase 33 did for its analogous
     finding) rather than silently fixing or silently ignoring; low cost to fix once the new
     function exists (just swap the call site), so leaning toward "fix it as a small bonus task"
     is reasonable, but this is the planner's/discuss-phase's call, not locked by CONTEXT.md.

2. **(RESOLVED — planner's discretion per CONTEXT.md; 34-03 chose paired `.gpx` + JSON
   `expected.json` with the tolerance in each language's shared assertion helper, matching the
   recommendation below.)** Exact on-disk corpus fixture format (JSON? YAML? a paired `.gpx` +
   `.expected.ts`/`.dart`?)**
   - What we know: D-01 locks the *location* (on disk, language-neutral) but CONTEXT.md
     explicitly marks the format as Claude's discretion.
   - What's unclear: Whether a plain JSON expected-values file (trivially read by both
     `JSON.parse` in Vitest and `jsonDecode` in Dart) is sufficient, or whether per-field
     tolerance annotations (D-03: distance/elevation ~1e-6 m, duration/points/bbox exact) need
     to be encoded in the fixture itself vs. hardcoded in each test's assertion helper.
   - Recommendation: A plain JSON file per fixture (`expected.json`) with the tolerance encoded
     once in each language's shared assertion helper (not per-fixture) is simplest and avoids
     inventing a fixture-specific DSL — but this is genuinely open for the planner to decide.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Dart SDK | Dart port implementation + tests | ✓ (verified) | 3.12.2 (stable) | — |
| Flutter SDK | `flutter test` runner | ✓ (verified) | 3.44.2 (stable) | — |
| Node.js | Vitest suite, OpenAPI regeneration | ✓ (verified) | v22.12.0 | — |
| `gpx` pub package | GPX parsing | ✓ (verified, installed at 2.3.0, matches pubspec constraint) | 2.3.0 | — |
| PocketBase / Go toolchain | `moving_duration` migration | Not directly probed this session (no `go`/`pocketbase` CLI check run) | — | Existing migration files are Go source compiled as part of the `db/` module's normal build; no new tooling needed beyond what already builds `db/main.go` |

No missing dependencies with no fallback. No missing dependencies with fallback needed — every
tool this phase touches is already present and verified in this environment.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No auth surface changes in this phase |
| V3 Session Management | No | — |
| V4 Access Control | No | The convert endpoint's existing auth posture (public, unauthenticated per its current implementation) is unchanged by PORT-04 — only its *output* changes |
| V5 Input Validation | Yes | GPX parsing must not crash or hang on malformed input. The pre-parse sanitize pass (Pattern 1) is itself the V5 control — it neutralizes 5 confirmed crash vectors before they reach `GpxReader`. On the server side, the convert endpoint already validates content-type branches and empty-body (`+server.ts:71-73`); PORT-04 must preserve that, changing only the success-path response shape |
| V6 Cryptography | No | No crypto surface in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Malformed/adversarial GPX crashing the parser (DoS on the client, or on the server's transcode path) | Denial of Service | Client: the sanitize pass (Pattern 1) + existing try/catch-and-toast pattern in `importTrailFile`/`_onFinish`/`_saveRecordedTrack` (all three already fail soft on any exception, per the codebase's established precedent — verified in this session by reading each). Server: `+server.ts`'s existing `try { gpx2trail(...) } catch { throw 400 }` pattern is unchanged by PORT-04 |
| A crafted GPX file with an extremely large point count causing excessive on-device compute (client-side DoS via battery/CPU) | Denial of Service | Out of scope for this phase per CONTEXT.md (no point-count limit is mentioned in any decision); worth flagging as a residual risk the corpus doesn't cover, not a phase blocker — real-world GPX tracks (even multi-day recordings) are bounded by GPS sample rate to a few hundred thousand points at most, and the existing `/valhalla/height`/`/valhalla/trace-route` call sites already chunk at 500 points |

## Sources

### Primary (HIGH confidence — read directly from source in this session)
- `web/src/lib/models/gpx/gpx-metrics-computation.ts` — full file read, the algorithm to port
- `web/src/lib/models/gpx/gpx.ts` — full file read, `getTotals()` assembly
- `web/src/lib/models/gpx/utils.ts` — full file read, `haversineDistance`
- `web/src/lib/models/gpx/waypoint.ts` — full file read, `ele`/`lat`/`lon` fallback semantics
- `web/src/lib/util/gpx_util.ts` — full file read, `gpx2trail`/`fromFile`
- `web/src/routes/api/v1/trail/convert/+server.ts` — full file read
- `web/src/lib/util/icon_util.ts` — `icons` array + `getIconForLocation`
- `~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/gpx_reader.dart` — full file read (installed package source, not docs)
- `~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/wpt.dart` — full file read
- `~/.pub-cache/hosted/pub.dev/geobase-1.5.0/lib/src/geodesy/spherical/spherical_great_circle.dart` — read, `distanceTo` formula
- `app/lib/util/gpx_util.dart`, `trail_import_util.dart`, `route_planner_handoff_util.dart`, `route_planner_screen.dart`, `navigation_screen.dart` (relevant sections), `navigation_stats_provider.dart`, `components/navigation/track_save_options_sheet.dart` — full/targeted reads
- `app/lib/models/trail.dart`, `app/lib/entities/trail_entity.dart`, `web/src/lib/models/trail.ts` — read for `moving_duration` propagation surface
- `db/migrations/1747064968_collections_snapshot.go`, `1780000006_trail_external_reference_provider_text.go`, `1778583263_updated_trails_bounding_box.go` — read for migration pattern and field IDs
- `web/openapi.config.js`, `web/scripts/generate-openapi.js`, `web/src/lib/models/api/openapi_schemas.ts` — read for OpenAPI regeneration mechanics
- `.planning/phases/33-conversion-correctness/33-CONTEXT.md`, `33-VERIFICATION.md` — read in full
- `.planning/phases/34-dart-conversion-port/34-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md` — read
- Empirical: direct `dart run`/`node` execution of probe scripts against the real installed `gpx` package and V8, this session, 2026-07-31 (see Code Examples)

### Secondary (MEDIUM confidence)
- None — every claim in this document was either read directly from source in this repo/package,
  or empirically executed and observed in this session.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; existing package versions confirmed via local pub cache
- Architecture: HIGH — every file/function referenced was read directly, not inferred
- Pitfalls: HIGH — the two highest-risk pitfalls (parser crashes, float agreement) were
  empirically reproduced/measured on this machine, not assumed from documentation or training data
- Security: MEDIUM — no dedicated threat-modeling tool run; based on direct code reading of
  existing error-handling patterns

**Research date:** 2026-07-31
**Valid until:** 30 days (stable domain — GPX 1.1 spec, installed package versions, and the ported
TS algorithm are all fixed; re-verify if `gpx` or `maplibre`/`geobase` pub versions bump, or if
Phase 33's algorithm changes again before this phase executes)
