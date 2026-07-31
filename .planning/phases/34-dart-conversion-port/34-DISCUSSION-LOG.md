# Phase 34: Dart Conversion Port - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-31
**Phase:** 34-Dart Conversion Port
**Areas discussed:** Shared fixture corpus format, Convert endpoint breaking change, Moving time semantics (CONV-06), On-device elevation correction

---

## Shared fixture corpus format

### Q1 — Relationship to Phase 33's locked D-05 ("inline fixtures, not disk files")

| Option | Description | Selected |
|--------|-------------|----------|
| Shared on-disk corpus, D-05 amended | Language-neutral corpus both Vitest and Dart read; one source of truth | ✓ |
| Keep D-05 — duplicate fixtures per language | Honors the locked decision but the two sets can drift by hand | |
| Generate the corpus from the TS implementation | Agreement by construction, but bakes TS bugs in as "expected" | |

**Notes:** The tension was surfaced explicitly — PORT-02 requires a corpus two languages read, which cannot be inline in both without reintroducing the drift the requirement exists to prevent.

### Q2 — How expected values are established (TS rejected as oracle)

| Option | Description | Selected |
|--------|-------------|----------|
| Hand-derived for defect cases, TS-derived + reviewed for the rest | Rigor on the known-answer CONV-01..05 cases; tractable elsewhere | ✓ |
| Hand-derived for every fixture | Maximum rigor; impractical for large realistic tracks | |
| TS-derived, then reviewed for all | Cheapest; makes the corpus a regression harness, not a contract | |

### Q3 — What "identical" means for floating-point values

| Option | Description | Selected |
|--------|-------------|----------|
| Tight absolute tolerance, documented per field | ~1e-6 m on distance/elevation; duration, counts, bbox exact | ✓ |
| Exact equality on every field | Strongest reading, but a 1-ULP trig difference would fail for no real defect | |
| Loose tolerance (0.01 m) | Easy to keep green; wide enough to hide a genuine divergence | |

**Notes:** `dart:math` and V8 trig are not required to agree bit-for-bit, so exact equality could fail for reasons indistinguishable from a real bug.

### Q4 — How much of Phase 33's internal structure Dart must replicate

| Option | Description | Selected |
|--------|-------------|----------|
| Public metrics only; skip cumulativeDistance | Corpus asserts what gpx2trail persists; Dart internals free | ✓ |
| Public metrics + the monotonic/final split | Protects a future on-device per-segment feature | |
| Full internal parity | Max drift protection; ports code with no Dart consumer | |

**Notes:** `cumulativeDistance`'s only consumer is the web crop slider (Phase 33 D-01/D-02) — no app equivalent. Flagged the real trap: Dart must reproduce the `final*` getter values, not the monotonic `total*Smoothed` values.

---

## Convert endpoint breaking change

### Q1 — Rollout strategy given app-store clients can't be force-updated

| Option | Description | Selected |
|--------|-------------|----------|
| Hard break — change the response in place | Narrow blast radius, fails loudly, kills the legacy compute path | ✓ |
| Version it — add /v2/trail/convert | Old builds keep working; legacy trail-computing path survives | |
| Content negotiation on the existing path | Same problem as versioning, behind a subtler switch | |

### Q2 — Where the reverse-geocode step goes

| Option | Description | Selected |
|--------|-------------|----------|
| App calls reverse-geocode separately when online | Keeps PORT-01 satisfied; offline `location` is simply empty | ✓ |
| Server returns GPX + location in a JSON envelope | Preserves today's UX; isn't really transcode-only | |
| Drop auto-fill; user always types it | Simplest; visible regression for file import | |

**Notes:** Found by reading the endpoint — it reverse-geocodes `trail.location` via Nominatim at `+server.ts:87-99`, which stripping to transcode-only would silently drop.

### Q3 — Success response format

| Option | Description | Selected |
|--------|-------------|----------|
| Raw GPX body, `application/gpx+xml` | Honest, cacheable, trivially consumed by the Dart gpx package | ✓ |
| JSON envelope `{ gpx: "..." }` | Uniform JSON handling; dishonest wrapper, escaping cost | |
| JSON in / JSON out throughout | Tighter contract; breaks the app's existing multipart upload | |

### Q4 — Whether the web frontend also switches

| Option | Description | Selected |
|--------|-------------|----------|
| Web keeps computing in-browser, unchanged | Smallest diff, no regression; both clients converge on the same shape | ✓ |
| Web routes everything through the transcode endpoint | Uniform, but adds a round trip to plain .gpx uploads that works offline today | |
| Decide during planning | Defer to the planner | |

---

## Moving time semantics (CONV-06)

### Q1 — How moving time is determined (free-text response)

| Option | Description | Selected |
|--------|-------------|----------|
| Elapsed minus pausedAccumSeconds | Explicit pauses only | |
| Auto-pause on a speed/distance threshold | Strava-style heuristic | |
| Both — explicit now, auto-pause behind a setting | | |

**User's choice:** *(free text)* "We already display the time in movement to the user. For a recording/navigation session this value is the trail's duration."

**Notes:** Verified in code — `navigation_stats_provider.dart`'s tick is a no-op while `isPaused || isStationary`, so `NavigationStats.elapsed` is already moving time and already on screen. No new derivation needed; all three proposed options were superseded. Two consequences were surfaced: the recording duration is session state rather than a function of the GPX (so the conversion needs an optional duration override), and CONV-06 therefore cannot be pinned by the GPX-in/metrics-out corpus.

### Q2 — How a recording's moving time survives a later web route edit

| Option | Description | Selected |
|--------|-------------|----------|
| Separate field — duration stays elapsed, add moving_duration | One meaning per field; updateTotals structurally cannot destroy it | ✓ |
| Provenance field — duration means moving time for recordings | Matches CONV-06's wording; one field with two meanings + a guard to remember | |
| Accept the loss, document it | Ships fastest; silent unrecoverable loss found by a hiker, not a test | |
| Defer to its own phase | Keeps 34 focused; knowingly ships the data-loss path | |

**Notes:** Raised by the user ("Does that lead to problems down the line, when a user edits and saves the trail on the web?"). Investigated rather than assumed: the load path does *not* recompute, but ~14 route-mutation call sites funnel into `updateTotals()` which overwrites all four metrics from the GPX. Moving time is unrecoverable once overwritten, and no provenance field exists on `Trail` in either model.

### Q3 — Which other stats the session supplies

| Option | Description | Selected |
|--------|-------------|----------|
| Only moving_duration from the session | Every other stat reproducible and corpus-covered; nothing shifts on edit | ✓ |
| Session supplies all stats it already tracks | Matches the live numbers the user watched; not reproducible from the GPX | |
| Session stats now, reconcile later | Knowingly ships two sources of truth for three numbers | |

**Notes:** Refines the user's earlier framing ("the recording provides the trail's stats and overrides the GPX values") down to the one value that genuinely cannot be derived from a GPX.

---

## On-device elevation correction

### Q1 — How correction works in the Dart path (free-text response)

| Option | Description | Selected |
|--------|-------------|----------|
| Correction stays a separate, optional step outside the conversion | Pure offline conversion; caller-driven correction | |
| Conversion auto-corrects when online, skips when offline | Output depends on connectivity | |
| Drop elevation correction from the app entirely | Simplest; worse recording elevation than web | |

**User's choice:** *(free text)* "Show these options only when online. When offline skip directly to trail_create_screen."

**Notes:** Takes the shape of option 1 (caller-driven, conversion stays pure) plus a connectivity gate on the UI. Surfaced that this also fixes a live defect — `route_planner_screen.dart:513-524` currently strands the user on the planner when offline.

### Q2 — Online behaviour and scope of the sheet (free-text response)

**User's choice:** *(free text)* "1. The online path should explicitly ask the user via the sheet. 2. importTrailFile should show the same sheet with snapToRoad and recalculate heights options."

**Notes:** `showTrackSaveOptionsSheet` already exists, already returns `(recalcHeights, followRoads)` both off by default, and is already used by the recording path. Extending it to planner + import makes it the single post-capture gate. Because both toggles default off, the offline skip is behaviourally identical to "declined both" — one code path.

---

## Claude's Discretion

- Where the Dart conversion module lives and whether it mirrors the TS file layout.
- The corpus's on-disk directory layout and expected-values file format.
- How PORT-03 ("`/trail/convert` called for none of them") is proven.
- What the route planner writes into `duration`.

## Deferred Ideas

- Auto-pause tuning / threshold-based moving time — existing stationary detection suffices.
- Recomputing already-stored trail metrics (CONV-F01) — still deferred from Phase 33.
- Richer web presentation of `moving_duration` beyond the display rule.
- `2026-07-31-trail-create-screen-offline-gaps.md` — reviewed, not folded; belongs to Phase 35.
