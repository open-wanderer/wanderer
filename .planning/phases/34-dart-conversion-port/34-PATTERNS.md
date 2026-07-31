# Phase 34: Dart Conversion Port - Pattern Map

**Mapped:** 2026-07-31
**Files analyzed:** 13
**Analogs found:** 11 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/util/gpx_conversion_util.dart` (NEW) | utility (pure transform) | transform | `web/src/lib/models/gpx/gpx-metrics-computation.ts` + `gpx.ts` (cross-language port source) | exact (algorithm), role-match (Dart utility structure via `app/lib/util/gpx_util.dart`) |
| `sanitizeGpxEleAndTime` (in same new file, or `gpx_util.dart`) | utility (pre-parse sanitize) | transform | `sanitizeGpxEmail` in `app/lib/util/gpx_util.dart:8-13` | exact |
| `fixtures/gpx-corpus/**/input.gpx` + `expected.json` (NEW, repo root) | fixture / config | batch (read-only corpus) | Phase 33's inline TS fixtures in `gpx-metrics-computation.test.ts` (`gpxXml`/`trkptXml` helpers) — no on-disk precedent exists, this is a deliberate new convention (D-01) | no analog (new convention) |
| `app/test/util/gpx_conversion_util_test.dart` (NEW) | test | transform | `app/test/util/gpx_util_test.dart` | exact |
| `web/src/lib/models/gpx/gpx-corpus.test.ts` or similar (NEW) | test | transform | `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` | exact |
| `app/lib/util/trail_import_util.dart` (MODIFIED — `convertGpxToTrail`) | utility (call-site swap: HTTP → local compute) | request-response → transform | itself (existing `importTrailFile`/`convertGpxToTrail`), pattern source for error/toast handling | exact |
| `app/lib/routes/route_planner_screen.dart` (MODIFIED — `_onFinish`, offline gate) | controller (screen) | event-driven | `navigation_screen.dart`'s `_saveRecordedTrack` (already implements the sheet + online-gate pattern this file needs to adopt) | exact |
| `app/lib/routes/navigation_screen.dart` (MODIFIED — moving_duration wiring) | controller (screen) | event-driven | itself, `_saveRecordedTrack` (:733) already the reference implementation | exact |
| `app/lib/components/navigation/track_save_options_sheet.dart` (MODIFIED — reuse, no shape change needed) | component (bottom sheet) | request-response (user confirm) | itself | exact — no analog needed, extend call sites only |
| `app/lib/util/gpx_util.dart` (MODIFIED — delete `GpxMappingUtils.getTotals()`) | utility | transform | itself | n/a (deletion) |
| `web/src/routes/api/v1/trail/convert/+server.ts` (MODIFIED — transcode-only) | route (API endpoint) | request-response | itself; response-shape precedent from other raw-file-returning routes not needed — pattern is same file, narrower behavior | exact |
| `db/migrations/*_add_moving_duration_to_trails.go` (NEW) | migration | CRUD (schema) | `db/migrations/1778583800_persist_trail_bounds.go` (simple additive `NumberField`, idempotent `GetByName` guard, `Down` reverses cleanly) | exact |
| Trail models (`web/src/lib/models/trail.ts`, `app/lib/models/trail.dart`, `app/lib/entities/trail_entity.dart`) + `openapi_schemas.ts` (MODIFIED) | model | CRUD | existing `duration` field in each of the four locations | exact |

## Pattern Assignments

### `app/lib/util/gpx_conversion_util.dart` (NEW utility, transform)

**Analog:** `web/src/lib/models/gpx/gpx-metrics-computation.ts` (port source) + `web/src/lib/models/gpx/gpx.ts`'s `getTotals()` (assembly) + existing Dart structural conventions from `app/lib/util/gpx_util.dart`.

**Do not name it `getTotals`** — collides in spirit with the existing buggy extension on `Gpx` in `gpx_util.dart`. Use a distinct name, e.g. `computeTrailMetrics(Gpx gpx, {double? movingDurationOverride})`.

**Algorithm to port verbatim** (`gpx-metrics-computation.ts:26-236`, full file already read — do not re-fetch):
- `parseElevation` semantics (lines 15-24): `undefined`/`null`/blank/non-numeric/`NaN`/`Infinity` → no data; genuine `0` is real.
- The defer-then-publish elevation state machine in `addAndFilter()` (lines 97-235) — port the full state machine, not a simplified accumulator (Pitfall 2/CONV-04 regression risk).
- Expose only `finalElevationGain`/`finalElevationLoss` (lines 74-81) as the Dart port's public elevation output — **not** `totalElevationGainSmoothed`/`LossSmoothed`. Do not port `cumulativeDistance` (D-04/Pitfall 3) — it has no Dart consumer.
- Thresholds: `GpxMetricsComputation(5, 5)` — same `(thresholdXY_m, thresholdZ_m)` as `gpx.ts:106`.

**Public-metrics assembly** (`gpx.ts:97-167`, `getTotals()`): bounding box + centroid loop starting at `i = 0` (not `i = 1` — CONV-01/CONV-02 fix, Pattern 4), `summedPointCount` for the centroid divisor (not `allPoints.length`), `totalDistance = metrics.totalDistanceSmoothed` (smoothed, not raw).

**Haversine — reuse, do not hand-roll:**
```dart
// Source: app/lib/util/gpx_util.dart:108-113 (existing usage pattern)
final calculator = SphericalGreatCircle(Geographic(lat: prev.lat!, lon: prev.lon!));
final dist = calculator.distanceTo(Geographic(lat: curr.lat!, lon: curr.lon!));
// distanceTo defaults radius: 6371000.0 — matches TS's R=6371km*1000 exactly
```

**Duration override** (D-13): accept an optional `Duration? movingDurationOverride` parameter; when absent, derive `duration` from `(lastTrkpt.time - firstTrkpt.time)` exactly as `gpx.ts:117-123` does; when present, that value populates the new `movingDuration` output field, `duration` still always comes from the GPX.

**Waypoint icon mapping (Pitfall 6)** — when building `Waypoint` objects directly (not via server JSON), replicate `gpx2trail`'s icon guard:
```typescript
// Source: web/src/lib/util/gpx_util.ts:48-50
if (wpt.sym && icons.includes(wpt.sym as typeof icons[number])) {
    wp.icon = wpt.sym as typeof icons[number];
}
```
```dart
// Dart equivalent — route through the existing lookup, do not assign the raw string:
// fontAwesomeIconsMap[wpt.sym] ?? FontAwesomeIcons.circle
// (app/lib/util/icon_util.dart:1018, app/lib/models/converter/fa_icon_data_converter.dart)
```

---

### `sanitizeGpxEleAndTime` (pre-parse sanitize pass)

**Analog:** `sanitizeGpxEmail` (`app/lib/util/gpx_util.dart:4-13`) — copy this exact pattern (doc comment explaining *why*, single `replaceAllMapped`/regex rewrite, called before `GpxReader().fromString()`).

```dart
// Source: app/lib/util/gpx_util.dart:8-13 — the precedent to mirror exactly
String sanitizeGpxEmail(String xml) {
  return xml.replaceAllMapped(
    RegExp(r'<email>([^@<]+)@([^<]+)</email>'),
    (m) => '<email id="${m[1]}" domain="${m[2]}"/>',
  );
}
```

New function must neutralize (per RESEARCH.md's empirically-verified crash list): empty `<ele></ele>`, whitespace-only `<ele>`, non-numeric `<ele>`, empty `<time></time>` — rewrite each to self-closing (`<ele/>`, `<time/>`) which `GpxReader` already treats as `null`. Chain both sanitize passes at every parse call site: `GpxReader().fromString(sanitizeGpxEleAndTime(sanitizeGpxEmail(xml)))`.

---

### Shared fixture corpus (`fixtures/gpx-corpus/`, NEW, repo root)

**No direct analog** — Phase 33's precedent is inline TS fixtures (`gpx-metrics-computation.test.ts`'s `gpxXml`/`trkptXml` helper functions), which D-01 explicitly amends for the cross-language case. Structure per RESEARCH.md's recommendation (Claude's discretion, D-01 constrains location only):
```
fixtures/gpx-corpus/
├── 01-two-point-segment/{input.gpx, expected.json}
├── 02-partial-elevation/...
├── 03-missing-vs-empty-ele/...   # THE landmine fixture — <ele></ele> present, canary for Pitfall 1
├── 04-switchback-scramble/...
├── 05-jittery-track/...
└── 06-multi-anchor-planned-route/...
```
`expected.json` field naming should mirror the TS `GPXFeature` shape (`gpx.ts:25-34`) minus `cumulativeDistance`/`hash` (D-04 public-metrics-only scope): `centroid`, `boundingBox`, `distance`, `elevationGain`, `elevationLoss`, `duration`, plus `waypoints`, `name`, `description`.

**Reader precedent (verified, no assets entry needed):**
```dart
// Verified: app/test — flutter test CWD = package root; relative File reads work with no
// pubspec.yaml assets: entry (asset-bundle restriction only applies to rootBundle.load()
// inside a running app, not flutter test/dart test)
final file = File('../fixtures/gpx-corpus/01-two-point-segment/input.gpx');
```
```typescript
// Vitest CWD = web/ (no root override in vite.config.ts)
fs.readFileSync(path.resolve(__dirname, '../../../../fixtures/gpx-corpus/01-two-point-segment/input.gpx'))
```

**Tolerance (D-03):** distance/elevation ~1e-6 m absolute; duration/point-counts/bbox exact. Encode the tolerance once in each language's shared assertion helper, not per-fixture (RESEARCH.md's recommendation).

---

### `app/test/util/gpx_conversion_util_test.dart` (NEW test)

**Analog:** `app/test/util/gpx_util_test.dart` (full file read above — copy its structure exactly).

**Imports pattern** (lines 1-4):
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:gpx/gpx.dart';
```

**Structure:** `group('functionName', () { test('description', () { ... }); });` nesting, doc-comment-style test names explaining the specific defect/edge case being pinned (e.g. `'501-point input is downsampled to ≤500 and last point appears exactly once'`), matching the existing table-driven-by-comment style — no dedicated table-driven runner package is used in this codebase; explicit named `test()` blocks are the convention.

---

### `app/lib/util/trail_import_util.dart` — `convertGpxToTrail` (MODIFIED)

**Analog:** itself — full file already read above.

**Current (to be replaced):**
```dart
// Source: app/lib/util/trail_import_util.dart:80-108
Future<Trail> convertGpxToTrail(WidgetRef ref, FormData formData) async {
  final res = await ref.read(apiProvider).post('/trail/convert', data: formData);
  // ... placeholder id/created/updated injection, Trail.fromJson(...)
}
```
Replace the HTTP POST body with a local call into `gpx_conversion_util.dart`'s `computeTrailMetrics`, preserving the placeholder-id/created/updated injection pattern for any locally-constructed `Trail`/`Waypoint` objects (this part of the function's shape is reusable — only the data source changes from server response to local computation).

**Error handling pattern to preserve** (`importTrailFile`, lines 30-78): `try { ... } catch (e) { showError(); }` — a toast-and-stay pattern, not a rethrow. `showError()` closure at lines 37-47 is the reusable error UI call.

---

### `app/lib/routes/route_planner_screen.dart` — `_onFinish` (MODIFIED, D-15/D-16)

**Analog:** `app/lib/routes/navigation_screen.dart`'s `_saveRecordedTrack` (lines ~700-760+, already read above) — this is the reference implementation the planner screen must be brought in line with.

**Pattern to copy — online gate + sheet:**
```dart
// Source: app/lib/routes/navigation_screen.dart:730-733 (structure to replicate)
Future<void> _saveRecordedTrack(BuildContext context) async {
  final options = await showTrackSaveOptionsSheet(context);
  if (options == null) return;
  if (_savingTrack) return;
  setState(() => _savingTrack = true);
  // ... (recalcHeights, followRoads) = options; conditional Valhalla calls
}
```
Route planner's existing dead-end at `_onFinish`/`finishPlanning` (`:513-524` per RESEARCH.md, catches offline failure and shows a toast, stranding the user) must instead check online status **before** showing the sheet, and skip straight to `trail_create_screen` when offline (D-15) — this mirrors D-16's fix. The `_finishing` boolean guard (already present, lines ~500-524 read above) is the correct existing double-tap guard pattern — keep it, just relocate the online check ahead of the HTTP-dependent branch.

**Error handling to preserve** (already correct, lines 501-524 above): `try { ... } catch (_) { toast } finally { setState(() => _finishing = false); }`.

---

### `app/lib/routes/navigation_screen.dart` — moving_duration wiring (MODIFIED)

**Analog:** itself; `NavigationStats.elapsed`/`_pausedAccum` (`app/lib/provider/navigation_stats_provider.dart:229-235`, per RESEARCH.md — already the moving-time source of truth per D-09, no new derivation needed). Wire `NavigationStats.elapsed` into the new `movingDurationOverride` parameter of `computeTrailMetrics` at the point `_saveRecordedTrack` currently builds its `Gpx` and calls the conversion.

---

### `app/lib/util/gpx_util.dart` — delete `GpxMappingUtils.getTotals()` (D-17)

**Current buggy code being deleted** (full extract above, lines 96-135): `for (int i = 1; ...)` loop (drops first point), no defer-then-publish filter, raw (not smoothed) distance. Consumers `elevation_profile.dart:178` and `trail_panel.dart:44` must be redirected onto the new `computeTrailMetrics` from `gpx_conversion_util.dart` instead — do not leave them calling a function that no longer exists, and do not leave a second buggy implementation in place per D-17's rationale.

---

### `web/src/routes/api/v1/trail/convert/+server.ts` (MODIFIED — transcode-only, D-05/D-06/D-07)

**Analog:** itself (full file read above, 108 lines).

**Keep unchanged:** the three input branches (multipart/JSON/raw-text, lines 47-68), the empty-body 400 guard (lines 71-73), the `handleError` wrapper (line 106, `import { handleError } from "$lib/util/api_util"`).

**Remove:**
```typescript
// Source: +server.ts:75-85 — gpx2trail call and Trail construction, REMOVE
// Source: +server.ts:87-100 — reverse-geocode step, REMOVE (moves to app per D-07)
// Source: +server.ts:102-104 — json(trail) response, REPLACE
```

**Replace success response with raw GPX** (D-06): after validating `gpxData` is non-empty, return it directly with `Content-Type: application/gpx+xml` instead of constructing/returning a `Trail`. Errors keep the existing `handleError`/`ClientResponseError` JSON shape — no change to error branches.

**JSDoc/OpenAPI block to update** (lines 8-39): the `responses.200` schema changes from `$ref: '#/components/schemas/Trail'` to a raw string/binary GPX response; per Pitfall 4, this endpoint's own JSDoc is only one of 6 `elevation_gain`-containing locations in `openapi_schemas.ts` (grep-verified at lines ~390, 467, 522, 627, 668, 692) — those 5 others need the new `moving_duration` field added independently (see Trail model section below), not this endpoint's shape change.

---

### `db/migrations/*_add_moving_duration_to_trails.go` (NEW)

**Analog:** `db/migrations/1778583800_persist_trail_bounds.go` (full file read above) — simplest precedent for "add a new additive `NumberField` to the `trails` collection, idempotent, clean `Down`".

**Pattern to copy:**
```go
// Source: db/migrations/1778583800_persist_trail_bounds.go:162-170
func addTrailBoundsField(collection *core.Collection, name string) {
	if collection.Fields.GetByName(name) != nil {
		return
	}
	collection.Fields.Add(&core.NumberField{
		Name: name,
	})
}
```
```go
// Structure to copy (lines 94-109, adapted): Up() finds the trails
// collection, adds the field, saves.
func init() {
	m.Register(func(app core.App) error {
		trailsCollection, err := app.FindCollectionByNameOrId("trails") // or "e864strfxo14pm4" per RESEARCH.md Pitfall 7
		if err != nil {
			return err
		}
		if trailsCollection.Fields.GetByName("moving_duration") == nil {
			// Mirror the existing `duration` field's exact shape (field id
			// ukr9rqz4: type number, min:0, max:null, onlyInt:false) but
			// required:false so absence is meaningful (D-10's "no value" state)
			if err := trailsCollection.Fields.AddMarshaledJSONAt(N, []byte(`{
				"hidden": false,
				"id": "<new-unique-id>",
				"max": null,
				"min": 0,
				"name": "moving_duration",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			}`)); err != nil {
				return err
			}
		}
		return app.Save(trailsCollection)
	}, func(app core.App) error {
		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		trailsCollection.Fields.RemoveByName("moving_duration")
		return app.Save(trailsCollection)
	})
}
```
Use `AddMarshaledJSONAt` (per `1780000006_...go:31` precedent) mirroring the exact `duration` field shape (`{"id": "ukr9rqz4", "type": "number", "min": 0, "max": null, "onlyInt": false}` per RESEARCH.md's Pitfall 7 verification) rather than the plain `&core.NumberField{}` shorthand used in the simpler bounds example, since `min:0`/`onlyInt:false` need to be explicit.

---

### Trail models — `moving_duration` propagation (MODIFIED, 4 files)

**Analog:** each file's own existing `duration` field — same file, sibling addition.

```typescript
// web/src/lib/models/trail.ts:31 — sibling addition
duration?: number;
moving_duration?: number;  // NEW
```
```dart
// app/lib/models/trail.dart:76 — sibling addition (freezed/json_serializable)
@Default(0) double duration,
@JsonKey(name: 'moving_duration') double? movingDuration,  // NEW
```
```dart
// app/lib/entities/trail_entity.dart:26 — sibling addition (ObjectBox)
double? duration;
double? movingDuration;  // NEW — wire fromModel/toModel mirroring existing `duration` handling exactly (entity constructor default at :90, fromModel mapping at :113, toModel at :161)
```
```typescript
// web/src/lib/models/api/openapi_schemas.ts — add moving_duration at all 6
// elevation_gain-adjacent JSDoc locations (grep-verified lines 390, 467, 522,
// 627, 668, 692), then `npm run openapi:generate` and diff
// static/docs/api/wanderer.openapi.json
```
After Dart model/entity edits: `dart run build_runner build --delete-conflicting-outputs` to regenerate `trail.freezed.dart`/`trail.g.dart`/ObjectBox model (expect incidental unrelated `.g.dart` churn from the same project-wide pass — not a regression, per STATE.md's Phase 27 precedent).

**Web display rule (D-10, not a new file — `web/src/routes/trail/edit/[id]/+page.svelte`):** show `moving_duration` when present else `duration`; the ~14 call sites funneling through `updateTotals()` must never write `moving_duration` — only read/display it.

## Shared Patterns

### Error handling — toast-and-stay (offline-tolerant)
**Source:** `app/lib/util/trail_import_util.dart:30-78` (`importTrailFile`), `app/lib/routes/route_planner_screen.dart:501-524` (`_onFinish`)
**Apply to:** All three capture-path call sites touched by this phase.
```dart
try {
  // ... conversion / save work
} catch (e) {
  // toast, no rethrow — leaves the user on the current screen to retry
  ref.read(toastProvider.notifier).add(ToastMessage(type: ToastType.error, ...));
} finally {
  if (mounted) setState(() => _finishing = false); // or equivalent guard flag
}
```

### Double-tap / concurrent-invocation guard
**Source:** `route_planner_screen.dart`'s `_finishing`, `navigation_screen.dart`'s `_savingTrack`
**Apply to:** Any new async screen action wired to `computeTrailMetrics`/the options sheet.
```dart
if (_finishing) return;
setState(() => _finishing = true);
try { ... } finally { if (mounted) setState(() => _finishing = false); }
```

### Pre-parse sanitize before every `GpxReader().fromString()`
**Source:** `sanitizeGpxEmail` (`app/lib/util/gpx_util.dart:4-13`)
**Apply to:** `gpx_conversion_util.dart`'s new `sanitizeGpxEleAndTime`, and every call site parsing non-self-generated GPX (`trail_import_util.dart:66`, any new recording/planner parse call).
```dart
GpxReader().fromString(sanitizeGpxEleAndTime(sanitizeGpxEmail(xml)))
```

### PocketBase additive-field migration
**Source:** `db/migrations/1778583800_persist_trail_bounds.go:162-170` (idempotent `GetByName` guard), `1780000006_...go:31-46` (`AddMarshaledJSONAt` for a fully-specified field including constraints)
**Apply to:** The new `moving_duration` migration.

### OpenAPI schema — 6 duplicated locations, not one $ref
**Source:** `web/src/lib/models/api/openapi_schemas.ts` (grep for `elevation_gain`: lines 390, 467, 522, 627, 668, 692)
**Apply to:** Any `Trail`-shaped schema field addition, including this phase's `moving_duration` and the convert endpoint's own response-shape JSDoc.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `fixtures/gpx-corpus/**` (on-disk corpus itself) | fixture/config | batch | New convention deliberately introduced by D-01, amending Phase 33's inline-fixture precedent; no existing on-disk cross-language fixture directory exists anywhere in the repo to copy structurally — layout is Claude's/planner's discretion per CONTEXT.md |
| Corpus reader helper shared between Vitest and `flutter test` | utility (test-support) | batch | No existing dual-language test-fixture reader exists; must be authored fresh in each language following each language's own idiomatic file-read (`fs.readFileSync`/`path.resolve` vs `dart:io` `File`), per RESEARCH.md's verified CWD behavior |

## Metadata

**Analog search scope:** `web/src/lib/models/gpx/`, `web/src/lib/util/gpx_util.ts`, `web/src/routes/api/v1/trail/convert/`, `app/lib/util/`, `app/lib/routes/route_planner_screen.dart`, `app/lib/routes/navigation_screen.dart`, `app/lib/components/navigation/`, `app/test/util/`, `db/migrations/`, `web/src/lib/models/trail.ts`, `app/lib/models/trail.dart`, `app/lib/entities/trail_entity.dart`, `web/src/lib/models/api/openapi_schemas.ts`
**Files scanned:** ~20 (all read directly, no inference)
**Pattern extraction date:** 2026-07-31
