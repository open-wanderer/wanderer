# Phase 21: Route Planner Handoff & Entry Point - Research

**Researched:** 2026-07-17
**Domain:** Flutter mobile (go_router navigation, Riverpod state, GPX synthesis, ObjectBox-backed Settings model)
**Confidence:** HIGH

## Summary

This phase wires the last piece of the v1.5 Route Planner milestone: a real entry point (replacing a TEMPORARY hardcoded route) and a real handoff (replacing nothing — HANDOFF-01 is net-new). Both directions are mechanically well-understood because they compose four already-built, already-tested pieces of this codebase almost unchanged: `TrailSourceSelectScreen`'s `_SourceActionCard` pattern (entry UI), `plannedGpxProvider`/`route_anchor_provider.dart` (route state, Phase 19/20), `pendingImportedTrail` + the `/trail/create/edit` route builder (handoff safety net, pre-existing from GPX import), and `ElevationTab`'s exact `/valhalla/height` fetch-and-merge code (Phase 20) — this phase's one-time elevation fetch is a straight copy-with-modification of that same logic, run once instead of debounced-on-tab-visibility.

The two genuinely new pieces are: (1) a `Behavior.allowAutoGeolocate` field ported onto the Flutter `Settings` freezed model (mirroring `web/src/lib/models/settings.ts`'s existing `Behavior` type) plus the matching `SettingsEntity` ObjectBox JSON-blob field, gating GPS resolution at planner entry; and (2) a category ID lookup for D-08's hike→category / bike→category pre-fill, which has **no existing reverse-mapping helper** — categories are operator-managed runtime data (no `hiking`/`biking` category IDs exist in any migration or seed file), so the mapping must be a best-effort substring match against the loaded `categoryProvider` list (mirroring `costingForCategory`'s own `.contains('bike')` heuristic), with graceful degradation (leave category unset) if no match is found.

**Primary recommendation:** Build the draft `Trail` via `Trail.empty().copyWith(...)` exactly as `trail_import_util.dart` does, populating `expand: TrailExpand(gpxData: GpxWriter().asString(mergedGpx), gpx: mergedGpx, waypointsViaTrail: [])` — `gpxData` (the raw XML string) is what `form_data_util.dart`'s `toFormData()` actually uploads on create (`isCreate && gpxData != null` → multipart `gpx` field), so the parsed `Gpx` object alone is not sufficient; both must be set.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Hike/bike entry dialog + route registration | Browser/Client (Flutter UI + go_router) | — | Pure client navigation/UI state, no server involvement |
| `allowAutoGeolocate` gate + GPS resolution | Browser/Client (Flutter) | Database/Storage (ObjectBox `SettingsEntity`) | Setting is fetched/cached client-side; gate check is a local read, no new API call |
| One-time elevation fetch at handoff | Browser/Client (Flutter) | API/Backend (`/api/v1/valhalla/height`, pre-existing, unchanged) | Client orchestrates timing (once, at Finish tap); backend endpoint already exists, untouched |
| Draft Trail construction + `pendingImportedTrail` handoff | Browser/Client (Flutter) | — | In-memory-only until the user taps Save on the create/edit screen; no persistence happens in this phase |
| Category pre-fill lookup | Browser/Client (Flutter) | Database/Storage (`categoryProvider`'s ObjectBox cache) | Reads already-cached category list; no new endpoint |

No capability in this phase touches the Go/PocketBase backend or SvelteKit web app — confirmed by `REQUIREMENTS.md`'s Out of Scope table ("Any Go backend or SvelteKit changes... already exist and require no changes").

## User Constraints (from CONTEXT.md)

<user_constraints>

### Locked Decisions

- **D-01:** Hike/bike selection is a modal bottom sheet with two `_SourceActionCard`-style tappable cards — not an `AlertDialog`, not a segmented control.
- **D-02:** The sheet is dismissible (back/tap-outside closes with no navigation) — never `isDismissible: false`.
- **D-03 (SCOPE ADD):** Port `allowAutoGeolocate` to the Flutter `Settings` model (`app/lib/models/settings.dart`), mirroring web's `Settings.behavior.allowAutoGeolocate`. Gate the planner's initial GPS-centering behind it. Scope: schema/model field + gate check only — **no new settings-screen toggle UI required**.
- Router replacement: `/route-planner`'s TEMPORARY hardcoded registration (`router_provider.dart:254-260`) and `TrailSourceSelectScreen`'s TEMPORARY direct-push "Plan a route" card (`trail_source_select_screen.dart:72-82`) are both replaced by the real flow: card → bottom sheet → resolved `initialCenter`/`travelProfile` → `/route-planner` push.
- **D-04:** "Finish planning" is an app-bar action on `RoutePlannerScreen`. Undo/redo relocate from the app bar into the top-right map `controls` Column (same slot as the auto-routing toggle).
- **D-05:** Handoff requires ≥2 route anchors; below that, Finish is disabled/blocked (mirrors Phase 20 D-13's <2-anchor empty-state precedent).
- **D-06:** If the one-time `/api/v1/valhalla/height` fetch fails at handoff, proceed without elevation and hand off silently — no error dialog/snackbar/retry UI (matches Phase 20 D-11's best-effort precedent).
- **D-07 (SCOPE CHANGE):** Route anchors never become `Waypoint` records. The draft Trail carries only the synthesized GPX track — `waypoints: []` — same shape as a plain GPX-file import with no named points. This overrides HANDOFF-01's original "synthesized GPX + named waypoints" wording; `REQUIREMENTS.md` and `ROADMAP.md` were amended in the same session.
- **D-08:** The hike/bike choice pre-fills the draft Trail's `category` field (Hike → hiking category, Bike → biking category) — the reverse of `costingForCategory`'s existing category→profile heuristic.

### Claude's Discretion

- Exact category ID/lookup logic for D-08's hike→category / bike→category mapping — no existing reverse-lookup helper found during scouting (confirmed again in this research pass: no `hiking`/`biking` category IDs exist in any DB migration or seed file — categories are operator-managed runtime data).
- Bottom sheet card copy/icons for the hike/bike dialog (D-01) — follow `_SourceActionCard`'s existing visual conventions; UI-SPEC has already locked exact copy (see below).
- Exact fallback value for `initialCenter` when `allowAutoGeolocate` is false or GPS fails (D-03) — e.g. last map camera position (`mapCameraProvider`) vs. a fixed default.
- Whether `allowAutoGeolocate` needs a settings-screen toggle in this phase, or ships model-only until a future settings phase surfaces it (defaults to "no new toggle UI required").

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. D-03's `allowAutoGeolocate` port and D-07's waypoint-scope correction are implementation-mechanism/requirement-wording clarifications of already-scoped HANDOFF requirements, reconciled directly in `REQUIREMENTS.md`/`ROADMAP.md` rather than left as loose ideas.

</user_constraints>

## Phase Requirements

<phase_requirements>

| ID | Description | Research Support |
|----|-------------|------------------|
| HANDOFF-01 | Finish planning hands off the route as a draft Trail (synthesized GPX track only, no Waypoint records; elevation populated via one-time `/valhalla/height` fetch at handoff) to the trail create/edit screen | `pendingImportedTrail` mechanism (verbatim reuse), `plannedGpxProvider` shape, `GpxWriter().asString()` for `expand.gpxData`, `ElevationTab`'s exact height-fetch/merge code as the copy-and-adapt source, `Trail.empty().copyWith(...)` construction pattern, `form_data_util.dart`'s `toFormData` requiring `gpxData` (not just parsed `gpx`) on create |
| HANDOFF-02 | Route Planner reachable from a new entry point in trail-source-select flow | `trail_source_select_screen.dart`'s existing (TEMPORARY) "Plan a route" `_SourceActionCard`, ready to have its `onTap` replaced |
| HANDOFF-03 | Hike/bike selection dialog before planner opens, sets initial fixed travel profile | `route_anchor_provider.dart`'s `travelProfile` family-argument-fixed design (already built to receive this value); `gpx_util.dart`'s `'pedestrian'`/`'bicycle'` string constants to reuse verbatim; `Geolocator`/`foregroundPositionStreamProvider` pattern for D-03's GPS gate |

</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Tech stack for this phase: Dart/Flutter only (`app/lib/**/*.dart`) — no SvelteKit or Go changes (confirmed also by `REQUIREMENTS.md`'s Out of Scope table).
- Naming: snake_case files, camelCase functions/variables, PascalCase classes/types, boolean fields prefixed `is`/`allow` (existing `allowAutoGeolocate` follows this).
- Riverpod 3.3.1 + `riverpod_annotation` codegen — explicitly type `AsyncValue`/listener closure params (project-wide convention; a prior bug (`quick-260712-pac`) came from an untyped `ref.listen` closure silently resolving `isLoading` etc. as `dynamic`).
- Memory note override: use `.value`, not `.valueOrNull`, for nullable `AsyncValue` access.
- Freezed/JSON codegen conventions: every new freezed class needs `part '<file>.freezed.dart'` + `part '<file>.g.dart'` and a `build_runner` regen (see Pitfall section).
- L10n keys: literal snake_case getters (e.g. `finish_disabled_hint`), added to `app/lib/i18n/app_en.arb` (the template arb) at minimum; other locale `.arb` files have historically been updated selectively (not always all locales in one commit) — English is the hard requirement for this phase.

## Standard Stack

### Core (all pre-existing — no new packages)

| Library | Version (installed) | Purpose | Why Standard (in this codebase) |
|---------|---------|---------|--------------|
| `gpx` | 2.3.0 `[VERIFIED: pubspec.yaml]` | GPX model + `GpxWriter().asString()` XML serialization | Already the sanctioned GPX type throughout `gpx_util.dart`, `elevation_tab.dart`, `planned_gpx_provider.dart` |
| `go_router` | pinned per pubspec (not re-verified this phase — unchanged since Phase 18) | Route registration, `extra` passing | Already the app's only router; `pendingImportedTrail` exists specifically to work around its `extra`-loss-on-refresh edge case |
| `flutter_riverpod` / `riverpod_annotation` | 3.3.1 (per CLAUDE.md) | State providers (`routeAnchorsProvider`, `plannedGpxProvider`, `settingsProvider`) | Established app-wide pattern; this phase adds no new provider *types*, only consumes existing ones plus one new `Behavior` field |
| `geolocator` | 13.0.2 (per CLAUDE.md) | GPS fix for D-03's gated initial-center resolution | Already used by `foreground_position_stream_provider.dart` and `map_screen.dart`'s identical "resolve GPS once, gated" pattern |
| `font_awesome_flutter` | 11.0.0 | `FontAwesomeIcons.personHiking`/`.bicycle`/`.check` | Pre-existing, already-mapped icons in `icon_util.dart`'s `getTrailIcon()` — UI-SPEC mandates reusing these exact icons |
| `freezed_annotation` | 3.1.0 | New `Behavior` class on the `Settings` model | Matches every other nested `Settings` type (`SettingsLocation`, `SettingsPrivacy`) |

### Alternatives Considered

None — every piece of this phase's stack is a direct reuse of an already-adopted library; no new dependency decision exists to make. `REQUIREMENTS.md`'s Out of Scope table explicitly rules out any backend/package changes for this phase.

**Installation:** None required — no new `pubspec.yaml` entries.

## Package Legitimacy Audit

Not applicable — this phase installs zero new external packages. Every library used (`gpx`, `go_router`, `flutter_riverpod`, `geolocator`, `font_awesome_flutter`, `freezed_annotation`) is already present in `app/pubspec.yaml` and was vetted in prior phases. The Package Legitimacy Gate protocol is skipped per its own scope condition ("whenever this phase installs external packages").

## Architecture Patterns

### System Architecture Diagram

```
TrailSourceSelectScreen ("Plan a route" card)
        │  onTap
        ▼
showModalBottomSheet (Hike/Bike cards, D-01/D-02)
        │  card tap → sheet closes (Navigator.pop)
        ▼
resolve initialCenter                                    resolve travelProfile
  ref.read(settingsProvider)                                'pedestrian' | 'bicycle'
    .behavior?.allowAutoGeolocate == true                   (from which card was tapped)
        │                    │
     true                  false/GPS-fails
        │                    │
  foregroundPositionStreamProvider   mapCameraProvider (last saved) or fixed fallback
  (first GPS fix, one-shot)          (implementer's discretion)
        │                    │
        └────────┬───────────┘
                  ▼
     context.push('/route-planner', ...)
                  │
                  ▼
        RoutePlannerScreen(travelProfile, initialCenter)
        (routeAnchorsProvider(travelProfile) family — fixed for session)
                  │  user taps/drags/inserts anchors (Phase 19/20, unchanged)
                  ▼
        plannedGpxProvider(travelProfile)  ── pre-elevation Gpx, live
                  │  user taps app-bar "Finish" (D-04), gated on ≥2 anchors (D-05)
                  ▼
   ┌─────────────────────────────────────────────┐
   │  Handoff sequence (new this phase)           │
   │  1. read plannedGpxProvider → points          │
   │  2. buildNavShape(points) → POST /valhalla/   │
   │     height (ONE-SHOT, not debounced)          │
   │  3. merge ele into a fresh Gpx (silent on      │
   │     failure — D-06)                            │
   │  4. GpxWriter().asString(mergedGpx) → XML       │
   │  5. Trail.empty().copyWith(                    │
   │       category: <D-08 lookup>,                 │
   │       expand: TrailExpand(gpxData: xml,         │
   │         gpx: mergedGpx, waypointsViaTrail: []))  │
   │  6. pendingImportedTrail = draftTrail            │
   │  7. navContext.push('/trail/create/edit',        │
   │       extra: draftTrail)                         │
   └─────────────────────────────────────────────┘
                  │
                  ▼
        TrailCreateScreen(trail: draftTrail)
        (existing screen, unchanged — reads expand.gpx for
         map preview, category pre-selected, waypoints
         section shows its normal empty state)
```

### Recommended File Changes

```
app/lib/
├── routes/
│   ├── trail_source_select_screen.dart   # "Plan a route" card onTap → showModalBottomSheet
│   ├── route_planner_screen.dart         # app-bar Finish action; undo/redo move to controls Column
│   └── trail_create_screen.dart          # unchanged — verify it already handles category-prefilled + empty-waypoints Trail correctly (it should, this is the plain-GPX-import shape)
├── provider/
│   └── router_provider.dart              # /route-planner real registration (remove TEMPORARY hardcode)
├── components/
│   └── route_planner/
│       └── travel_profile_sheet.dart     # NEW — the D-01/D-02 hike/bike bottom sheet content widget
├── util/
│   ├── gpx_util.dart                     # ADD: categoryForTravelProfile(travelProfile, categories) reverse-lookup helper (D-08)
│   └── route_planner_handoff_util.dart   # NEW (suggested) — the one-time-elevation-fetch + draft-Trail-build sequence, mirroring elevation_tab.dart's _fetchHeights/_buildEleMergedGpx, callable from route_planner_screen.dart's Finish action
└── models/
    └── settings.dart                     # ADD: Behavior freezed class + Settings.behavior field (D-03)
app/lib/entities/
    └── settings_entity.dart              # ADD: behaviorJson field + fromModel/toModel mapping (D-03)
```

### Pattern 1: Reuse `pendingImportedTrail` verbatim, do not build a parallel mechanism

**What:** `app/lib/util/trail_import_util.dart:28` declares `Trail? pendingImportedTrail` — a bare global, not a provider, specifically because go_router's `extra` can be silently dropped on a same-process router refresh (its doc comment explains the exact JSON-encode failure mode). `router_provider.dart:262-278`'s `/trail/create/edit` route builder already reads `state.extra is Trail` first, clears `pendingImportedTrail` on use, and falls back to it otherwise.

**When to use:** Any time code needs to hand a client-constructed, unsaved `Trail` to the create/edit screen. This phase is the second producer of this contract (the GPX-file-import flow being the first) — no changes to `router_provider.dart`'s `/trail/create/edit` builder are needed.

**Example:**
```dart
// Source: app/lib/util/trail_import_util.dart:109-111 (existing, GPX-import flow)
pendingImportedTrail = trail;
if (!navContext.mounted) return;
navContext.push('/trail/create/edit', extra: trail);
```
This phase's handoff should call the identical two lines with its own draft `Trail`.

### Pattern 2: One-time elevation fetch — copy `ElevationTab`'s fetch/merge code, drop the debounce/visibility gating

**What:** `app/lib/components/route_planner/elevation_tab.dart`'s `_fetchHeights()` (lines 85-113) and `_buildEleMergedGpx()` (lines 115-135) are the exact reference implementation: downsample via `buildNavShape(points)` (`gpx_util.dart`), `POST /valhalla/height` with `{'shape': shape}`, parse `response.data['height'] as List`, zip heights back onto the shape-aligned points (not the original untouched point list — index alignment matters), build a fresh `Gpx` with `ele` set per point.

**When to use:** At Finish-tap time, once, against the final `plannedGpxProvider(travelProfile)` snapshot — not gated on tab visibility, not debounced (there's no further edit to debounce against; the route is final).

**Example:**
```dart
// Source: app/lib/components/route_planner/elevation_tab.dart:85-113 (adapt: remove debounce/visibility checks, run once)
final gpx = ref.read(plannedGpxProvider(travelProfile));
final points = gpx.allPoints;
final shape = buildNavShape(points); // gpx_util.dart

Gpx finalGpx = gpx; // fallback: pre-elevation, if fetch fails (D-06 — silent)
try {
  final response = await ref.read(apiProvider).post('/valhalla/height', data: {'shape': shape});
  final heights = (response.data['height'] as List).cast<num>();
  finalGpx = _buildEleMergedGpx(shape, heights); // same builder as ElevationTab's
} catch (_) {
  // D-06: proceed silently with pre-elevation Gpx, no error UI
}
```

### Pattern 3: Draft-Trail construction — `Trail.empty().copyWith(...)`, `gpxData` (XML string) is the field that actually uploads

**What:** `app/lib/models/trail.dart:145-146` provides `Trail.empty()` (`id: ''`, `name: ''`, `created`/`updated`: `DateTime.now()`). `app/lib/util/form_data_util.dart:47-50` shows `isCreate && gpxData != null` is the condition that attaches a multipart `gpx` file from `expand.gpxData` (the **raw XML string**, not the parsed `Gpx` object) — this is what the server actually receives on `PUT /trail/form`. The parsed `expand.gpx` (`Gpx`, `includeFromJson/ToJson: false` on the model) exists purely for **client-side map rendering** before save (exactly as `trail_import_util.dart:101-107` parses `gpxData` back into `expand.gpx` after the server round-trip).

**When to use:** Building this phase's draft Trail. Both fields must be set:

```dart
// Pattern composed from: app/lib/models/trail.dart:145 (Trail.empty),
// app/lib/util/trail_import_util.dart (copyWith/expand shape),
// pub.dev gpx 2.3.0 GpxWriter.asString (XML serialization)
final xml = GpxWriter().asString(finalGpx);
final draftTrail = Trail.empty().copyWith(
  category: resolvedCategoryId, // D-08, may be null if no match found
  expand: TrailExpand(
    gpxData: xml,
    gpx: finalGpx,
    waypointsViaTrail: const [], // D-07 — never populated from anchors
  ),
);
```

### Pattern 4: Category reverse-lookup — best-effort substring match, no static ID exists

**What:** `costingForCategory` (`gpx_util.dart:22-29`) maps category-name → Valhalla costing string via `.contains('bike')`/`'cycling'`/`'bicycle')`. No inverse helper exists, and **no `hiking`/`biking` category ID is seeded anywhere** — categories are fully operator-managed runtime content (confirmed by grepping every `db/migrations/*.go` file; zero hits for category-seed data of any kind, unlike e.g. `1747061259_seed_actors.go` which does seed actors).

**When to use:** D-08's hike→category / bike→category pre-fill. Recommended approach — a new `categoryForTravelProfile(String travelProfile, List<Category> categories)` helper in `gpx_util.dart` (co-located with `costingForCategory`, its inverse):

```dart
// Suggested, not found in codebase — Claude's Discretion per 21-CONTEXT.md
String? categoryForTravelProfile(String travelProfile, List<Category> categories) {
  final wantsBike = travelProfile == 'bicycle';
  bool matches(Category c) {
    final name = c.name.toLowerCase();
    final short = (c.shortName ?? '').toLowerCase();
    final hay = '$name $short';
    return wantsBike
        ? (hay.contains('bike') || hay.contains('cycling') || hay.contains('bicycle'))
        : (hay.contains('hik') || hay.contains('walk') || hay.contains('foot'));
  }
  return categories.firstWhereOrNull(matches)?.id; // null if operator has no matching category — degrade gracefully, leave unset
}
```
Read categories from `ref.read(categoryProvider).value ?? []` (already cached by the time the user reaches the create/edit screen in the common case, but may be loading/empty on a cold start — a `null` result is a safe, silent degrade to "no pre-fill", matching a plain GPX import's default state per the UI-SPEC).

### Pattern 5: `allowAutoGeolocate` gate — mirror `map_screen.dart`'s existing gated-GPS-resolution shape

**What:** `map_screen.dart:105-136` already implements "resolve GPS once for initial camera, but only if nothing else already decided the center" using `foregroundPositionStreamProvider`. This phase's D-03 adds one more gate in front of that: check `Settings.behavior?.allowAutoGeolocate == true` before even subscribing.

**Example:**
```dart
// Source pattern: app/lib/routes/map_screen.dart:105-136 (adapt: add allowAutoGeolocate gate)
final settings = ref.read(settingsProvider); // per project convention, .value not .valueOrNull if AsyncValue-wrapped
if (settings?.behavior?.allowAutoGeolocate == true) {
  // subscribe to foregroundPositionStreamProvider, take first fix, resolve initialCenter
} else {
  // fall back — mapCameraProvider's last saved position, or a fixed default (implementer's discretion)
}
```

### Anti-Patterns to Avoid

- **Writing `ele` back into `plannedGpxProvider`:** Phase 20's D-10/D-11 deliberately keep that provider pre-elevation so it never re-fires Valhalla on every anchor edit. This phase's elevation merge must stay local to the handoff sequence — never call `plannedGpxProvider`'s notifier or otherwise mutate its output.
- **Converting `RouteAnchor`s into `Waypoint` records:** D-07 is an explicit, user-confirmed correction of the original requirement wording. `waypointsViaTrail` must be `const []`/omitted, never populated from `state.anchors`.
- **Setting only `expand.gpx`, not `expand.gpxData`:** the parsed `Gpx` object is `@JsonKey(includeFromJson: false, includeToJson: false)` — it never serializes to JSON and is not what `toFormData()` uploads. Only `gpxData` (raw XML string) reaches the server on create. Missing this produces a draft Trail that *renders* correctly in the create/edit screen's map preview but silently uploads with no track on save.
- **A `static` field for the handoff's one-shot elevation-fetch state:** Phase 19's own post-mortem (19-04 UAT bug) found a `static final RouteSegmentLayer()` field caused segments to silently stop rendering on a second screen-open because static state survived across mounts while native map state did not. Any handoff-sequence bookkeeping (e.g. an in-flight flag) must be instance-scoped, not static.
- **Reintroducing a settings-screen toggle for `allowAutoGeolocate`:** explicitly out of scope per D-03 ("no new toggle UI required"). Don't add one speculatively.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Passing an unsaved Trail across a route boundary | A new provider/singleton for "pending planner trail" | `pendingImportedTrail` (`trail_import_util.dart`) | Already solves the exact go_router `extra`-loss-on-refresh edge case; a second parallel mechanism would double the surface area for the same bug class |
| GPX XML serialization | Manual XML string-building | `GpxWriter().asString(gpx)` (`gpx` package 2.3.0) | Already a direct dependency; hand-rolled XML risks the same `<email>` non-standard-form class of bug `sanitizeGpxEmail` exists to patch |
| Elevation fetch/merge | A new Valhalla-height-calling helper from scratch | Copy-and-adapt `ElevationTab._fetchHeights`/`_buildEleMergedGpx` (`elevation_tab.dart`) | Already handles downsampling via `buildNavShape`, index-alignment between shape and heights, and the exact request/response shape Valhalla expects |
| GPS resolution gated by a setting | A new location-permission/stream wrapper | `foregroundPositionStreamProvider` (already handles permission request, service-disabled, and stream lifecycle) | Reinventing this reintroduces the permission-dialog-fires-more-than-once risk the existing ghost-subscription pattern was built to avoid |

**Key insight:** Every "don't hand-roll" item in this phase already has a working, tested implementation elsewhere in the same codebase from Phase 19/20 — the work here is composition and one small reverse-lookup (category), not new infrastructure.

## Common Pitfalls

### Pitfall 1: Forgetting `expand.gpxData` (raw XML), shipping only `expand.gpx` (parsed object)

**What goes wrong:** The create/edit screen's map preview renders perfectly (it reads `expand.gpx`), but saving produces a trail with no track — `toFormData()`'s `isCreate && gpxData != null` check silently skips the multipart `gpx` file.
**Why it happens:** `expand.gpx` is the natural field to reach for since it's what every other map-rendering code path in this app already consumes; `gpxData` is easy to forget because it's a "server round-trip" field in the GPX-import flow, not obviously required for a client-synthesized trail.
**How to avoid:** Always call `GpxWriter().asString(finalGpx)` and set both `gpxData` and `gpx` together, per Pattern 3 above.
**Warning signs:** A manual test where the map preview looks correct on the create/edit screen but the saved trail (after tapping Save) has zero distance/no track on the detail screen.

### Pitfall 2: Elevation merge index misalignment

**What goes wrong:** `_buildEleMergedGpx` zips `heights[i]` onto `shape[i]`, not onto the original unsampled `points[i]` — if the handoff code accidentally merges against `points` instead of the downsampled `shape`, elevations attach to the wrong coordinates once `points.length > 500` (Valhalla's shape cap via `buildNavShape`).
**Why it happens:** Both `points` (from `gpx.allPoints`) and `shape` (from `buildNavShape(points)`) look interchangeable at a glance but diverge exactly when a route exceeds 500 points.
**How to avoid:** Follow `ElevationTab`'s existing code verbatim — merge against `shape`, never against `points` directly (see its own doc comment: "merge the response back against this exact shape (not the full original point list) so index alignment holds").
**Warning signs:** Elevation profile/chart looks visually "shifted" or noisy specifically on long routes, not on short ones.

### Pitfall 3: Undo/redo relocation breaking the "instance field, not static" rule established in Phase 19

**What goes wrong:** If undo/redo's `IconButton`s are wrapped in a new reusable widget moved into the top-right `controls` Column and that widget (or any state it holds) is declared `static`, it risks the exact class of bug 19-04's UAT found for `RouteSegmentLayer` — stale state surviving a screen re-mount while native/local state resets.
**Why it happens:** Moving widgets between layout slots sometimes tempts factoring them into a separate `StatelessWidget`/`StatefulWidget` with a `static` cache for convenience.
**How to avoid:** Undo/redo are just `IconButton`s reading `state.undoStack`/`state.redoStack` and calling `notifier.undo`/`.redo` — no local state of their own, so this risk is low, but keep any new wrapper widget's fields non-static per the existing project-wide caution.
**Warning signs:** Undo/redo buttons work correctly the first time the planner screen opens in a session, then stop responding (or respond to stale data) on a second open.

### Pitfall 4: New `Behavior` freezed field requires a full codegen regen, and `SettingsEntity`'s JSON-blob pattern must be replicated exactly

**What goes wrong:** Adding `Behavior? behavior` to `Settings` (freezed) without running `build_runner` leaves `settings.freezed.dart`/`settings.g.dart` stale — `Settings.fromJson`/`copyWith` won't know about the new field, causing silent data loss on any settings round-trip. Separately, `SettingsEntity` stores every nested settings type as a `String? <field>Json` (see `locationJson`, `privacyJson`, `notificationsJson`) — a new `behaviorJson` field must follow that exact pattern (JSON-encode on `fromModel`, JSON-decode on `toModel`), not a different storage strategy.
**Why it happens:** Freezed/json_serializable/ObjectBox codegen files are easy to forget when adding a field mid-session; the existing `SettingsEntity` pattern is easy to diverge from if a different (e.g. flattened bool column) approach seems simpler.
**How to avoid:** Run `dart run build_runner build --delete-conflicting-outputs` (or the project's established regen command) after editing `settings.dart`; mirror `privacyJson`'s exact null-check/encode/decode shape for `behaviorJson`.
**Warning signs:** `flutter analyze` failures referencing missing `behavior` getter/param on `_$Settings`/`_$SettingsFromJson`; or (if codegen ran but the entity wasn't updated) settings that "reset" `allowAutoGeolocate` to null after an app restart because it was never persisted to ObjectBox.

### Pitfall 5: `costingForCategory`'s substring heuristic and D-08's reverse lookup can disagree with each other

**What goes wrong:** If an operator names a category e.g. "Mountain Biking Trails" (contains "bik" but also "hik" via "Hiking Trails" naming collisions are unlikely, but a category like "Bikepacking & Hiking" could match both directions), the forward (`costingForCategory`) and reverse (D-08's new helper) heuristics could disagree on the same category, producing a confusing "picked Hike but pre-filled a category that later maps back to bicycle costing" loop if the user later edits the category.
**Why it happens:** Both are independent substring heuristics over free-text operator-managed category names with no shared canonical vocabulary.
**How to avoid:** Not a hard requirement to reconcile (out of this phase's scope — the pre-fill is a convenience, the user can always change it on the create/edit screen), but the reverse-lookup helper should check `'bike'`/`'cycling'`/`'bicycle'` first (bicycle branch) exactly mirroring `costingForCategory`'s own check order, so the two heuristics are at least symmetric for any category name that would trip either one.
**Warning signs:** None expected to surface as a bug per se — flag as a known limitation if it comes up in manual testing.

## Code Examples

### Building the draft Trail at handoff (composed pattern, not found verbatim in codebase)

```dart
// Source: composed from app/lib/util/trail_import_util.dart (pendingImportedTrail
// mechanism), app/lib/models/trail.dart:145 (Trail.empty), app/lib/util/gpx_util.dart
// (buildNavShape), app/lib/components/route_planner/elevation_tab.dart (fetch/merge
// pattern), pub.dev gpx 2.3.0 GpxWriter
Future<void> finishPlanning(WidgetRef ref, BuildContext navContext, String travelProfile) async {
  final gpx = ref.read(plannedGpxProvider(travelProfile));
  final points = gpx.allPoints;
  if (points.length < 2) return; // D-05 guard, should already be enforced by disabled button

  final shape = buildNavShape(points);
  var finalGpx = gpx; // fallback: pre-elevation (D-06)
  try {
    final response = await ref.read(apiProvider).post('/valhalla/height', data: {'shape': shape});
    final heights = (response.data['height'] as List).cast<num>();
    finalGpx = _buildEleMergedGpx(shape, heights); // same shape as ElevationTab's private helper
  } catch (_) {
    // silent best-effort, D-06 — no toast/dialog
  }

  final categories = ref.read(categoryProvider).value ?? const [];
  final categoryId = categoryForTravelProfile(travelProfile, categories); // D-08, may be null

  final xml = GpxWriter().asString(finalGpx);
  final draftTrail = Trail.empty().copyWith(
    category: categoryId,
    expand: TrailExpand(gpxData: xml, gpx: finalGpx, waypointsViaTrail: const []),
  );

  pendingImportedTrail = draftTrail;
  if (!navContext.mounted) return;
  navContext.push('/trail/create/edit', extra: draftTrail);
}
```

### Entry-point route registration (replacing the TEMPORARY block)

```dart
// Source: app/lib/provider/router_provider.dart:254-260 (current TEMPORARY registration
// to be replaced with real travelProfile/initialCenter, supplied by the caller via `extra`
// or by pushing with named parameters once the hike/bike sheet resolves them)
GoRoute(
  path: '/route-planner',
  builder: (context, state) {
    final extra = state.extra as Map<String, dynamic>?; // or a dedicated typed extra class
    return RoutePlannerScreen(
      travelProfile: extra?['travelProfile'] as String? ?? 'pedestrian',
      initialCenter: extra?['initialCenter'] as ml.Geographic? ?? const ml.Geographic(lat: 0, lon: 0),
    );
  },
),
```
Note: the exact `extra` shape (typed record/class vs. `Map`) is implementer's discretion — `/map`'s existing route builder (`router_provider.dart:136-155`) shows the `Map<String, dynamic>` pattern already in use for optional typed extras in this router, a reasonable precedent to follow.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `/route-planner` hardcodes `travelProfile: 'pedestrian'` + `Geographic(lat:0, lon:0)` | Real values resolved from the hike/bike sheet + gated GPS/fallback | This phase (removes the Phase 19 TEMPORARY marker) | Planner now opens at a sensible location instead of `(0,0)` (null island) |
| `TrailSourceSelectScreen`'s "Plan a route" card pushes `/route-planner` directly | Card opens the hike/bike bottom sheet first | This phase | Matches HANDOFF-03's requirement; the TEMPORARY comment in `trail_source_select_screen.dart:77-80` explicitly names Phase 21 as the phase that replaces it |
| Route Planner has no way to leave with a result | App-bar Finish action produces a draft Trail via `pendingImportedTrail` | This phase (net-new, HANDOFF-01) | Closes the loop the whole v1.5 milestone was building toward |

**Deprecated/outdated:** The `router_provider.dart:250-253` comment block itself documents its own removal condition ("Remove this route once Phase 21 wires the real one") — this phase should delete that comment along with the hardcoded values, not just edit the values in place.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No category ID/short_name is reserved for "hiking"/"biking" anywhere in the DB — confirmed by grepping all `db/migrations/*.go` files, zero hits | Pattern 4 / Don't Hand-Roll | If an operator's instance happens to have differently-named categories with no "hik"/"bik"/"cycl" substrings at all, D-08's pre-fill silently no-ops (degrades to unset) — this is the documented, accepted fallback, not a crash risk, but worth flagging to the planner as expected/normal behavior, not a bug |
| A2 | `Settings.behavior` defaults to `false` when absent, mirroring web's `page.data.settings.behavior?.allowAutoGeolocate ?? false` pattern (`web/src/routes/settings/map/+page.svelte:27`) | Pattern 5 / D-03 | If the Flutter side instead defaults an absent field to `true`, the planner would start auto-centering on GPS for users who never opted in on web — a privacy-relevant behavior mismatch between platforms |
| A3 | `Trail.empty().copyWith(...)` is the correct base for a client-synthesized (never-uploaded) Trail, matching the shape `trail_import_util.dart` produces after its server round-trip (same `id: ''`, fresh `created`/`updated`) | Pattern 3 | If the create/edit screen or `trailSaveProvider.createTrail` has an undocumented assumption baked in from always receiving a *server-round-tripped* trail (e.g. expecting `distance`/bounds fields pre-populated), this could produce a trail that saves successfully but displays a zero/blank stat somewhere until the next server fetch recomputes it — low risk, cosmetic only |

## Open Questions

1. **Exact `extra` shape for `/route-planner`'s real route registration**
   - What we know: the router already has a precedent (`/map`'s `Map<String, dynamic>` extra) for optional typed values passed via `push(..., extra: {...})`.
   - What's unclear: whether a typed record (`(String, ml.Geographic)`) or a dedicated small class is preferred over a raw `Map` for this call site — the codebase uses both patterns elsewhere (`/trail/:id/navigate` uses a typed record tuple).
   - Recommendation: Claude's Discretion at plan time; either is consistent with existing precedent, a typed record edges out `Map` for type-safety and matches the more recently-added `/trail/:id/navigate` route.

2. **Where the `categoryForTravelProfile` reverse-lookup helper should live**
   - What we know: `costingForCategory` (its forward counterpart) lives in `gpx_util.dart`.
   - What's unclear: whether co-locating the reverse lookup there (despite `gpx_util.dart` not otherwise touching `Category` types — it would need a new import) is preferable to a new small `category_travel_profile_util.dart`, or inlining directly in the handoff sequence.
   - Recommendation: co-locate in `gpx_util.dart` next to `costingForCategory` for discoverability (documented as its inverse), unless the planner judges the new `Category` import there undesirable, in which case a small dedicated util file is an equally valid alternative.

## Environment Availability

No external tool/service/runtime dependencies beyond what's already installed and working in this codebase — this phase adds no new package, no new backend endpoint, and no new third-party service. `/api/v1/valhalla/height` (used for the one-time elevation fetch) is the same pre-existing endpoint Phase 20's `ElevationTab` already calls successfully; confirmed unchanged per `REQUIREMENTS.md`'s Out of Scope table ("`/api/v1/valhalla/route` and `/api/v1/valhalla/height` already exist and require no changes").

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `geolocator` (GPS permission/stream) | D-03's gated initial-center resolution | Yes (already a direct dependency, used by `foreground_position_stream_provider.dart`) | 13.0.2 | N/A — already required by existing map screens |
| `/api/v1/valhalla/height` (backend) | One-time elevation fetch at handoff | Yes (already called successfully by `ElevationTab`) | Unchanged | D-06: silent best-effort, hand off pre-elevation on failure |
| `gpx` package `GpxWriter` | Serializing the handoff `Gpx` to XML for `expand.gpxData` | Yes (already a direct dependency) | 2.3.0 | N/A |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None — every dependency in this phase is already present and already exercised by prior-phase code paths.

## Validation Architecture

Skipped — `workflow.nyquist_validation` is explicitly `false` in `.planning/config.json`.

(Note for the planner: existing Flutter unit tests relevant to this phase's composed logic already exist and should be extended, not newly scaffolded: `app/test/provider/planned_gpx_provider_test.dart`, `app/test/provider/route_anchor_provider_test.dart`, `app/test/util/gpx_util_test.dart`. A new test file for the handoff sequence — e.g. `app/test/util/route_planner_handoff_util_test.dart` or wherever the plan places the extracted handoff logic — is a natural home for asserting D-05's ≥2-anchor guard, D-06's silent-failure fallback, D-07's empty-waypoints invariant, and D-08's category-lookup fallback-to-null behavior, even though this section is formally skipped per config.)

## Security Domain

`security_enforcement` is `true` (ASVS level 1, block on `high`) in `.planning/config.json`, so this section is included per protocol, though this phase's actual attack surface is minimal — it is a client-only feature with no new API endpoints, no new authentication/session logic, and no new server-side input validation surface (the existing `/trail/form` and `/valhalla/height` endpoints, and their existing Zod/PocketBase-rule validation, are unchanged).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No new auth surface — trail creation already goes through the existing authenticated `/trail/form` endpoint, unchanged by this phase |
| V3 Session Management | No | No session changes |
| V4 Access Control | No | No new access-control decision — a user can only hand off/create a trail authored by themselves, exactly as the existing create/edit screen already enforces |
| V5 Input Validation | Marginal | The new `Behavior.allowAutoGeolocate` boolean field on `Settings` should use the same `z.boolean()`-equivalent typing discipline the web schema already applies (`web/src/lib/models/api/settings_schema.ts:26`) — on the Flutter side this is a typed freezed field (compile-time validated), no runtime validation library is used client-side in this codebase, consistent with existing `Settings` fields |
| V6 Cryptography | No | No new secrets/tokens/crypto introduced |
| V8 Data Protection (informal — not a numbered ASVS L1 category but relevant) | Yes | GPS location is only read locally to resolve a map center and is never transmitted anywhere by this phase (no new network call carries device location) — the existing `foregroundPositionStreamProvider`'s permission-request flow (via `geolocator`) is the sanctioned mechanism, already used by `map_screen.dart`/`navigation_screen.dart`; this phase must not bypass it (e.g. no direct `Geolocator.getCurrentPosition()` call outside the shared provider) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Silent GPS auto-centering without user consent | Information Disclosure (mild — local-only, no transmission) | D-03's `allowAutoGeolocate` gate is exactly this mitigation — default `false`/opt-in, matching web's existing settings default |
| Client-forged `category`/`gpxData` on trail create | Tampering | Already mitigated server-side — `/trail/form` (PocketBase collection rules + Go validation, unchanged by this phase) does not trust client-supplied `category` blindly beyond what any other create-trail path already allows; this phase does not introduce a new bypass, it uses the exact same `trailSaveProvider.createTrail` call path as manual trail creation |

## Sources

### Primary (HIGH confidence — direct codebase reads this session)

- `app/lib/routes/trail_source_select_screen.dart` — entry-point card pattern, TEMPORARY marker
- `app/lib/util/trail_import_util.dart` — `pendingImportedTrail` mechanism, doc-commented rationale
- `app/lib/provider/router_provider.dart` — `/route-planner` TEMPORARY registration, `/trail/create/edit` builder
- `app/lib/provider/planned_gpx_provider.dart`, `app/lib/provider/route_anchor_provider.dart` — route state shape, `travelProfile` family fixed-for-session design
- `app/lib/util/gpx_util.dart` — `costingForCategory`, `buildNavShape`, `buildGpxFromPoints`, `sanitizeGpxEmail`
- `app/lib/routes/route_planner_screen.dart` — current app-bar/controls layout (undo/redo location to be moved)
- `app/lib/components/route_planner/route_anchor_sheet.dart` (+ uncommitted local diff via `git diff`) — tabbed sheet header restructure, unrelated to this phase's scope but confirmed non-conflicting
- `app/lib/components/route_planner/elevation_tab.dart` — exact fetch/merge/debounce pattern to adapt for the one-time handoff fetch
- `app/lib/models/settings.dart`, `app/lib/entities/settings_entity.dart` — current `Settings`/`SettingsEntity` shape, confirmed `behavior` field absent
- `web/src/lib/models/settings.ts`, `web/src/lib/models/api/settings_schema.ts` — `Behavior` type and Zod schema to mirror
- `web/src/routes/settings/map/+page.svelte` — confirms `allowAutoGeolocate ?? false` default-to-false convention
- `app/lib/models/trail.dart` — `Trail.empty()`, `TrailExpand` shape (`gpxData` vs. `gpx` distinction)
- `app/lib/util/form_data_util.dart` — confirms `gpxData` (not `gpx`) is what `toFormData()` uploads on create
- `app/lib/provider/trail/trail_save_provider.dart` — `createTrail` call path, waypoint-creation loop (confirms empty `waypointsViaTrail` is a trivial no-op)
- `app/lib/routes/trail_create_screen.dart` — category resolution on save (`CategoryPicker.resolve`), confirms `trail.category` is a plain `String?` id
- `app/lib/util/category_icon_util.dart`, `app/lib/util/icon_util.dart` — confirms `FontAwesomeIcons.personHiking`/`.bicycle` are pre-existing, already-mapped icons per UI-SPEC's mandate
- `app/lib/provider/trail/category_provider.dart` — confirms categories are fetched/cached, no static hiking/biking IDs
- `app/lib/provider/foreground_position_stream_provider.dart`, `app/lib/routes/map_screen.dart` (lines 60-136) — GPS-gating precedent for D-03
- `app/lib/models/route_anchor.dart` — `RouteAnchor`/`RouteSegment`/`RouteAnchorsSnapshot` shapes (confirms D-07's "distinct in-memory type, never a Waypoint")
- `/Users/christianbeutel/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/gpx_writer.dart` — confirms `GpxWriter().asString(gpx)` API signature
- `db/migrations/*.go` (grep) — confirms zero seeded `hiking`/`biking` category IDs anywhere in the schema/seed history
- `.planning/config.json` — `nyquist_validation: false`, `security_enforcement: true` (ASVS level 1)

### Secondary / Tertiary

None used — every claim in this research was verified directly against the actual codebase in this session; no WebSearch was needed since this phase is pure composition of already-adopted, already-documented internal patterns.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, every library already in `pubspec.yaml` and exercised by working code
- Architecture: HIGH — every pattern (entry sheet, handoff, elevation fetch, GPS gate) has a direct, working precedent already in this codebase from Phase 19/20 or `map_screen.dart`
- Pitfalls: HIGH — sourced from this codebase's own documented bug history (19-04 UAT static-field bug, `form_data_util.dart`'s gpxData-vs-gpx distinction) rather than generic external advice
- Category reverse-lookup (D-08): MEDIUM — confirmed no existing helper/static ID exists, but the exact matching heuristic is a genuine design choice with no single "correct" answer, flagged as Claude's Discretion per CONTEXT.md

**Research date:** 2026-07-17
**Valid until:** No external expiry driver (no new dependencies, no third-party API version to go stale) — revalidate only if a future phase changes `costingForCategory`, `Settings`, or the `pendingImportedTrail` mechanism.
