# Requirements: Wanderer Trail Navigation — v1.8 Offline Recording & Deferred Upload

**Defined:** 2026-07-31
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

**Milestone goal:** A hiker who records a trail with no signal can save it, review it, and fill in
its details on the spot — and it uploads itself when the phone next has a connection, without the
hiker doing anything.

**Design record:** `.planning/notes/offline-recording-deferred-upload-design.md`
**Open decisions for discuss-phase:** `.planning/research/questions.md`

---

## v1 Requirements

### Conversion Correctness (shared TS + Dart)

Fixes to the shared GPX→trail metrics computation, applied to `web/src/lib/models/gpx/gpx.ts`,
`web/src/lib/models/gpx/gpx-metrics-computation.ts`, and `web/src/lib/util/gpx_util.ts`. Each was
found by auditing the TS before porting it.

- [x] **CONV-01**: A track segment's first point is included in distance, bounding box, and centroid — a 2-point segment reports its real length instead of zero, and a planned route no longer cuts the corner at every anchor
- [x] **CONV-02**: The centroid divides by the same point count it summed, instead of summing `n − k` points and dividing by `n`
- [x] **CONV-03**: Points with no elevation are excluded from gain/loss instead of being treated as 0 m, so a partially-elevated GPX no longer reports a phantom plunge to sea level and back
- [x] **CONV-04**: Elevation gain/loss is sampled independently of the horizontal threshold, so a steep climb with little horizontal movement (switchbacks, scrambles) is measured rather than skipped
- [x] **CONV-05**: Distance is computed from the smoothed accumulator rather than the raw haversine sum, so GPS jitter no longer inflates it; the dead, misaligned `cumulativeDistance` array is removed. **Superseded 2026-08-01** by `.planning/quick/260801-opr-report-raw-distance-instead-of-the-5m-ga/`, scoped to the smoothed-distance half only (the `cumulativeDistance` rebuild stands): the 5 m gate chord-shortcuts switchbacks at real GPS sampling density, and FIT ground truth (`19440058502_ACTIVITY.fit`, `session.total_distance` = 10912.01 m) measured the raw accumulator at +0.54% against the gate's −3.29%. Distance is now the raw accumulator.
- [x] **CONV-06**: A trail recorded in the app reports moving time, excluding accumulated pause time; an imported file continues to report elapsed time

### Dart Conversion Port

- [x] **PORT-01**: The app computes a draft trail's name, description, waypoints, start coordinates, date, distance, elevation gain/loss, duration, and bounding box from a GPX entirely on-device, with no network call
- [x] **PORT-02**: A shared fixture test proves the Dart and TypeScript implementations produce the same metrics for the same GPX inputs, covering the CONV-01..05 defect cases explicitly
- [x] **PORT-03**: Recordings, route-planner output, and `.gpx` file imports all produce their trail through the Dart path — `POST /trail/convert` is no longer called for any of them
- [x] **PORT-04**: `POST /api/v1/trail/convert` transcodes kml/kmz/tcx/fit to GPX and returns it, without computing a trail; the published OpenAPI description matches the new behaviour
- [x] **PORT-05**: Importing a kml/kmz/tcx/fit file online still produces a correct trail, with the app measuring the server-transcoded GPX itself

### Local-First Unsynced Trails

**Source-agnostic by design.** An *unsynced trail* is one captured on-device that has never
reached the server, whether it came from ending a recording or from importing a GPX file with
no connection. These were originally worded "recording"-only, which left Phase 36's own goal
("a hiker who records a trail **or uploads a GPX** with no signal can save it") uncarried by
any requirement: Phase 35's OFFUI-03 gets an offline GPX import as far as a populated
`trail_create_screen`, and without this widening, pressing Save there would still fail — after
the hiker had filled in title, description, category and photos.

- [x] **REC-01**: Capturing a trail with no connection saves it — whether ended from a recording or imported from a GPX file — and the hiker is never shown a save failure caused by being offline
- [x] **REC-02**: A saved unsynced trail appears in the hiker's own-trails list (`/profile/<handle>/trails`) immediately, before it has ever reached the server — **not** in the Library, which is exclusively trails the hiker downloaded
- [x] **REC-03**: An unsynced trail is visibly distinguishable from a synced trail, and from a trail downloaded for offline use
- [x] **REC-06**: With no connection, the hiker's own-trails list still renders — showing every not-yet-uploaded trail plus those downloaded trails the hiker authored themselves — and states plainly that it is currently showing only what is available offline
- [x] **REC-04**: An unsynced trail survives app restart and stays associated with the account that captured it; signing in as a different account does not show or upload it, and signing out does not delete it
- [x] **REC-05**: A hiker can open, review, and edit an unsynced trail's details (title, description, category, photos) while still offline

### Background Upload

- [x] **SYNC-01**: An unsynced trail uploads on its own once the app is foregrounded with a working connection, with no action from the hiker
- [x] **SYNC-02**: Upload progress and failure are visible on the trail itself, inline in the hiker's own-trails list, rather than in a separate pending-uploads screen
- [x] **SYNC-03**: A hiker can manually retry an upload that failed or stalled
- [x] **SYNC-04**: An interrupted upload does not produce a duplicate trail on the server when it is retried
- [x] **SYNC-05**: Once uploaded, an unsynced trail becomes an ordinary trail — it keeps its identity in the library rather than appearing a second time

### Offline Create/Import UX

- [x] **OFFUI-01**: The map in the trail create/edit screen renders from downloaded regions when there is no connection, instead of going blank
- [x] **OFFUI-02**: Tag entry works with no connection — autocomplete degrades to showing nothing instead of erroring, and a typed tag still reaches the saved trail
- [x] **OFFUI-03**: Importing a `.gpx` file works with no connection
- [x] **OFFUI-04**: Attempting to import a kml/kmz/tcx/fit file with no connection explains that those formats need a connection and that GPX works offline, rather than showing a generic failure

---

## Future Requirements

Deferred. Tracked but not in this milestone's roadmap.

### Conversion

- **CONV-F01**: Backfill corrected metrics onto trails saved before v1.8 — no migration path exists today, and there is no meaningful install base to migrate
- **CONV-F02**: Moving time for imported GPX files — would require inferring pauses from timestamps rather than reading recorded pause state

### Recordings

- **REC-F01**: Queue a non-GPX file import for transcoding on reconnect, so imports complete eventually like recordings do
- **REC-F02**: Surface a warning when signing out with recordings still undrained

---

## Out of Scope

| Feature | Reason |
|---------|--------|
| Porting kml/kmz/tcx/fit transcoders to Dart | Depend on vendored `toGeoJSON`, JSZip, and `fit-parser`; no recording can reach them, so the cost buys only offline import of rare formats |
| Migrating already-saved trails to corrected metrics | `PUT /trail/form` stores client values and never recomputes; app is pre-production |
| Web UI changes | The app-only boundary still holds for web frontend work — only the shared conversion logic is in scope |
| Backward compatibility for `/api/v1/trail/convert`'s old response | The Flutter app is its only caller and the endpoint is not deployed in production anywhere |
| Caching tags in ObjectBox for offline autocomplete | Decided against in favour of degrading to no suggestions; free-form tags still work and are resolved at upload |
| Phase 31 on-device pass, Phase 29 VERIFICATION.md, Flutter dark mode | Carried v1.7 gaps and unrelated debt; deliberately excluded to keep the milestone focused |

---

## Traceability

Populated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CONV-01 | Phase 33 | Complete |
| CONV-02 | Phase 33 | Complete |
| CONV-03 | Phase 33 | Complete |
| CONV-04 | Phase 33 | Complete |
| CONV-05 | Phase 33 | Superseded 2026-08-01 (quick-260801-opr) |
| CONV-06 | Phase 34 | Complete |
| PORT-01 | Phase 34 | Complete |
| PORT-02 | Phase 34 | Complete |
| PORT-03 | Phase 34 | Complete |
| PORT-04 | Phase 34 | Complete |
| PORT-05 | Phase 34 | Complete |
| REC-01 | Phase 36 | Complete |
| REC-02 | Phase 36 | Complete |
| REC-03 | Phase 36 | Complete |
| REC-04 | Phase 36 | Complete |
| REC-05 | Phase 36 | Complete |
| REC-06 | Phase 36 | Complete |
| SYNC-01 | Phase 36 | Complete |
| SYNC-02 | Phase 36 | Complete |
| SYNC-03 | Phase 36 | Complete |
| SYNC-04 | Phase 36 | Complete |
| SYNC-05 | Phase 36 | Complete |
| OFFUI-01 | Phase 35 | Complete |
| OFFUI-02 | Phase 35 | Complete |
| OFFUI-03 | Phase 35 | Complete |
| OFFUI-04 | Phase 35 | Complete |

**Coverage:**

- v1 requirements: 25 total
- Mapped to phases: 25
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-31*
*Last updated: 2026-07-31 after v1.8 ROADMAP.md created (Phases 33-36, 25/25 requirements mapped)*
