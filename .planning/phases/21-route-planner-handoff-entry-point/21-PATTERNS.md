# Phase 21: Route Planner Handoff & Entry Point - Pattern Map

**Mapped:** 2026-07-17
**Files analyzed:** 8 (new + modified)
**Analogs found:** 8 / 8 (all internal, same-repo analogs — no external pattern needed)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/components/route_planner/travel_profile_sheet.dart` (NEW) | component (modal sheet) | request-response (tap → resolve → navigate) | `app/lib/routes/trail_source_select_screen.dart` (`_SourceActionCard`, lines 106-180ish) | exact |
| `app/lib/routes/trail_source_select_screen.dart` (MODIFY `onTap` at ~72-82) | controller/route | request-response | itself (existing `_importGpx`/card wiring, lines 33-56, 71-82) | exact |
| `app/lib/provider/router_provider.dart` (MODIFY `/route-planner` registration, lines ~254-262; reuse `/trail/create/edit` builder lines 262-278 unchanged) | route/config | request-response | itself — `/trail/create/edit` builder is the exact target contract to satisfy | exact |
| `app/lib/routes/route_planner_screen.dart` (MODIFY app bar actions + controls Column) | controller/component | event-driven (user tap) | itself — existing app-bar `actions` list + `controls` `Positioned` Column (undo/redo currently in app bar, auto-routing toggle already in controls Column) | exact |
| `app/lib/util/route_planner_handoff_util.dart` (NEW, suggested) | utility (orchestration) | request-response + file-I/O-like (GPX build) | `app/lib/components/route_planner/elevation_tab.dart` (`_fetchHeights`/`_buildEleMergedGpx`) + `app/lib/util/trail_import_util.dart` (`importTrailFile`, `pendingImportedTrail`) | exact (composed from two analogs) |
| `app/lib/util/gpx_util.dart` (ADD `categoryForTravelProfile`) | utility (transform) | transform | itself — `costingForCategory` (forward mapping), same file | exact (inverse of existing function) |
| `app/lib/models/settings.dart` (ADD `Behavior` freezed class + `behavior` field) | model | CRUD (config load/save) | itself — existing nested settings types (`SettingsLocation`, `SettingsPrivacy`) | exact |
| `app/lib/entities/settings_entity.dart` (ADD `behaviorJson` field) | model (ObjectBox entity) | CRUD | itself — existing `locationJson`/`privacyJson` fields + `fromModel`/`toModel` mapping | exact |

## Pattern Assignments

### `app/lib/components/route_planner/travel_profile_sheet.dart` (NEW component, request-response)

**Analog:** `app/lib/routes/trail_source_select_screen.dart`

**Imports pattern** (lines 1-8 of analog):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/trail_import_util.dart';
```
New sheet will additionally need `package:wanderer/models/...` for `Geographic` (maplibre) and provider imports for `settingsProvider`/`categoryProvider`/`foregroundPositionStreamProvider` per the GPS-gate pattern.

**Core card widget to copy verbatim (shape, not content)** — `_SourceActionCard` (lines 106-180):
```dart
class _SourceActionCard extends StatelessWidget {
  final FaIconData icon;
  final String title;
  final String description;
  final VoidCallback? onTap;
  final bool isLoading;

  const _SourceActionCard({
    required this.icon,
    required this.title,
    required this.description,
    this.onTap,
    this.isLoading = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final disabled = onTap == null && !isLoading;
    final resolvedBgColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.05)
        : theme.colorScheme.secondaryContainer.withValues(alpha: 0.4);
    final resolvedIconColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.38)
        : theme.colorScheme.primary;
    // ... resolvedTitleColor / resolvedDescriptionColor / resolvedBorderColor identical pattern
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: resolvedBorderColor),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: resolvedBgColor, borderRadius: BorderRadius.circular(12)),
                child: FaIcon(icon, color: resolvedIconColor),
              ),
              const SizedBox(width: 16),
              // Title + description Column ...
            ],
          ),
        ),
      ),
    );
  }
}
```
UI-SPEC mandates reusing this exact card shape (16px card radius, 12px icon-badge radius, `secondaryContainer` @ 40% alpha bg, `theme.colorScheme.primary` icon tint) for the two Hike/Bike cards, stacked with 8px gap inside a `showModalBottomSheet` (`isDismissible: true`, `enableDrag: true`, `BorderRadius.vertical(top: Radius.circular(20))` — the exact constant every other sheet in the app uses, per `trail_quick_filter_bar.dart`).

**Sheet trigger/dismiss pattern** — `showModalBottomSheet` convention referenced in UI-SPEC (`trail_quick_filter_bar.dart` et al., not re-read here since shape is given verbatim in UI-SPEC lines 87-93): `isDismissible: true`, no forced choice, `Navigator.pop(context)` on card tap before resolving `initialCenter`/pushing `/route-planner`.

---

### `app/lib/routes/trail_source_select_screen.dart` (MODIFY, controller/route)

**Analog:** itself — existing "Plan a route" card wiring.

**Current code to replace** (lines 71-82):
```dart
_SourceActionCard(
  icon: FontAwesomeIcons.route,
  title: l10n.trail_source_planner,
  description: "Design your perfect route from scratch using our map tools.",
  // TEMPORARY: routes to the test-only /route-planner entry point
  // (see router_provider.dart) for Phase 19 manual device
  // verification. Phase 21 (HANDOFF-02/03) replaces this with the
  // real hike/bike profile dialog.
  onTap: _importing ? null : () => context.push('/route-planner'),
),
```
**Pattern to follow for the replacement:** mirror `_importGpx`'s existing async-with-loading-flag shape (lines 33-56) — `setState(() => _importing = true)` before the async GPS/settings resolution, `finally { setState(() => _importing = false) }`, matching `_SourceActionCard`'s `isLoading` trailing-spinner convention (UI-SPEC explicitly calls out reusing this exact `isLoading` treatment rather than inventing a new spinner). Replace the `onTap` body with a call to `showModalBottomSheet` for the new hike/bike sheet; TEMPORARY comment block should be deleted (per RESEARCH's State-of-the-Art note), matching `router_provider.dart`'s own instruction to delete its paired TEMPORARY comment.

---

### `app/lib/provider/router_provider.dart` (MODIFY `/route-planner` registration)

**Analog:** itself.

**Current TEMPORARY code (lines 250-262, to delete/replace):**
```dart
// TEMPORARY test-only entry point for Phase 19 manual device
// verification — the real entry point (hike/bike dialog supplying
// travelProfile/initialCenter) is Phase 21's HANDOFF-02/03 scope.
// Remove this route once Phase 21 wires the real one.
GoRoute(
  path: '/route-planner',
  builder: (context, state) => RoutePlannerScreen(
    travelProfile: 'pedestrian',
    initialCenter: const Geographic(lat: 0, lon: 0),
  ),
),
```
**Target shape** — mirror the already-established `Map<String, dynamic>` typed-extra precedent used elsewhere in this same router (e.g. `/map`'s route builder, per RESEARCH's Code Examples section), reading `travelProfile`/`initialCenter` off `state.extra` instead of hardcoding.

**Downstream builder — read-only, do not modify** (lines 262-278, the `/trail/create/edit` contract this phase's handoff must satisfy):
```dart
GoRoute(
  path: '/trail/create/edit',
  builder: (context, state) {
    final extra = state.extra;
    if (extra is Trail) {
      pendingImportedTrail = null;
      return TrailCreateScreen(trail: extra);
    }
    final pending = pendingImportedTrail;
    if (pending != null) return TrailCreateScreen(trail: pending);
    return const TrailSourceSelectScreen();
  },
),
```

---

### `app/lib/util/route_planner_handoff_util.dart` (NEW, orchestration utility)

**Analog 1 (elevation fetch/merge):** `app/lib/components/route_planner/elevation_tab.dart`

**Imports pattern** (lines 1-9):
```dart
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/util/gpx_util.dart';
```

**Core fetch/merge pattern to copy-and-adapt (drop debounce/tab-visibility gating — run once):**
```dart
final gpx = ref.read(plannedGpxProvider(travelProfile));
final points = gpx.allPoints;
final shape = buildNavShape(points); // gpx_util.dart

Gpx finalGpx = gpx; // fallback: pre-elevation, if fetch fails (D-06 — silent)
try {
  final response = await ref.read(apiProvider).post('/valhalla/height', data: {'shape': shape});
  final heights = (response.data['height'] as List).cast<num>();
  finalGpx = _buildEleMergedGpx(shape, heights); // merge against shape, NOT points — index alignment
} catch (_) {
  // D-06: proceed silently with pre-elevation Gpx, no error UI
}
```
**Critical pitfall carried from analog:** merge heights against `shape` (the `buildNavShape`-downsampled array), never against the original `points` — they diverge once `points.length > 500` (Valhalla's shape cap).

**Analog 2 (handoff/navigation mechanism):** `app/lib/util/trail_import_util.dart`

**`pendingImportedTrail` global + push pattern** (lines 20-28 doc-comment context, plus the push call):
```dart
/// Fallback for the last imported [Trail] when GoRouter's `extra` doesn't
/// survive a same-process router refresh...
Trail? pendingImportedTrail;
```
```dart
// existing usage pattern (trail_import_util.dart, GPX-import flow)
pendingImportedTrail = trail;
if (!navContext.mounted) return;
navContext.push('/trail/create/edit', extra: trail);
```
This phase's handoff builds a draft `Trail` via `Trail.empty().copyWith(category: categoryId, expand: TrailExpand(gpxData: xml, gpx: finalGpx, waypointsViaTrail: const []))` (composed pattern, see RESEARCH.md Pattern 3), then calls the identical two lines above with its own `draftTrail`. **Do not build a parallel mechanism** — reuse `pendingImportedTrail` verbatim.

**Error handling pattern:** no try/catch needed around the navigation itself (matches `trail_import_util.dart`'s style — errors are only caught around the network/parse step, via `showError()` + `toastProvider`, not reused here since D-06 requires *silent* failure for elevation specifically).

---

### `app/lib/util/gpx_util.dart` (ADD `categoryForTravelProfile`, transform)

**Analog:** `costingForCategory` in the same file (lines 22-29, forward direction) — read for signature/style consistency; new function is its explicit inverse, co-located in the same file per RESEARCH's Open Questions recommendation.

**Pattern (composed, not existing verbatim — Claude's Discretion per CONTEXT.md D-08):**
```dart
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
  return categories.firstWhereOrNull(matches)?.id; // null if no match — degrade gracefully
}
```
Check order (`bike`/`cycling`/`bicycle` first) must mirror `costingForCategory`'s own check order for symmetry (Pitfall 5 in RESEARCH.md).

---

### `app/lib/models/settings.dart` (ADD `Behavior` freezed class + field)

**Analog:** existing nested settings types on the same `Settings` freezed class (`SettingsLocation`, `SettingsPrivacy` — same file, lines ~94-104 per RESEARCH.md's Integration Points).

**Pattern:** new `Behavior` freezed class with `bool? allowAutoGeolocate` (default resolves to `false` when absent, per web's `?? false` convention at `web/src/lib/models/settings.ts:56` and `web/src/routes/settings/map/+page.svelte:27`), added as a `behavior` field on `Settings`. Requires `part '<file>.freezed.dart'` + `part '<file>.g.dart'` regen via `build_runner` (Pitfall 4).

**Web analog to mirror shape from:**
- `web/src/lib/models/settings.ts:56` — `Behavior` type definition
- `web/src/lib/models/api/settings_schema.ts` — Zod schema shape

---

### `app/lib/entities/settings_entity.dart` (ADD `behaviorJson` field)

**Analog:** existing `locationJson`/`privacyJson` fields on the same entity — same file, same JSON-blob-per-field pattern (ObjectBox stores each nested settings type as a `String?` JSON blob, encoded on `fromModel`, decoded on `toModel`).

**Pattern:** add `String? behaviorJson`, mirror the exact null-check/encode/decode shape used by `privacyJson` (not a flattened boolean column — must match existing storage strategy per Pitfall 4).

---

### `app/lib/routes/route_planner_screen.dart` (MODIFY app bar + controls Column)

**Analog:** itself — existing app-bar `actions` (holds undo/redo currently, per RESEARCH lines 144-162) and existing top-right `controls` `Positioned(top: 128, right: 0)` Column (currently holds only the auto-routing toggle, per Phase 20 D-04 wiring).

**Current class-level caution to carry forward** (from the file's own doc comment, non-static field pattern):
```dart
// Instance field, NOT static: RouteSegmentLayer tracks whether its
// source/layers have been added to the CURRENT native style via its own
// `_added` flag. A `static` field would share that flag (and go stale)
// across every mount of this screen...
```
Any new wrapper widget introduced when moving undo/redo into the `controls` Column must follow this same non-static-field discipline (Pitfall 3).

**Target change:** app-bar `actions` list swaps undo/redo `IconButton`s for a single Finish `IconButton` (`FontAwesomeIcons.check`, 18px, `IconButton.styleFrom(backgroundColor: Theme.of(context).colorScheme.surface)`, disabled when `< 2` anchors per D-05, tooltip = `l10n.finish` when enabled / new "Add at least 2 anchors to finish your route." copy when disabled). Undo/redo move into the `controls` Column below the auto-routing toggle, keeping their exact Phase 19 pill styling unchanged (`theme.canvasColor` background, `BorderRadius.circular(24)`, `Padding(all: 8.0)`, `VisualDensity.compact`, 18px `FontAwesomeIcons.arrowRotateLeft`/`arrowRotateRight`) — position only changes.

---

## Shared Patterns

### Handoff navigation (`pendingImportedTrail`)
**Source:** `app/lib/util/trail_import_util.dart` (lines 20-28, 109-111 equivalent usage)
**Apply to:** `route_planner_handoff_util.dart`'s Finish-tap sequence — build draft `Trail`, set `pendingImportedTrail`, `navContext.push('/trail/create/edit', extra: draftTrail)`. Do not invent a parallel state-passing mechanism.

### GPS gating behind a setting
**Source:** `app/lib/routes/map_screen.dart` (lines 60-136, referenced in RESEARCH — gated GPS-resolution shape via `foregroundPositionStreamProvider`)
**Apply to:** the entry-point sheet's `initialCenter` resolution — check `Settings.behavior?.allowAutoGeolocate == true` (via `.value`, not `.valueOrNull`, per project memory convention) before subscribing to `foregroundPositionStreamProvider`; else fall back to `mapCameraProvider`'s last position or a fixed default.

### Silent best-effort network fetch (no error UI)
**Source:** `app/lib/components/route_planner/elevation_tab.dart`'s `_fetchHeights`/try-catch shape, reused per D-06/Phase 20 D-11 precedent.
**Apply to:** the one-time `/valhalla/height` fetch in the handoff sequence — catch, degrade to pre-elevation `Gpx`, never show a toast/dialog.

### `_SourceActionCard`-family visual consistency
**Source:** `app/lib/routes/trail_source_select_screen.dart` lines 106-180
**Apply to:** the two new Hike/Bike cards in `travel_profile_sheet.dart` — same 16px/12px radii, same `secondaryContainer`@40%-alpha badge background, same `theme.colorScheme.primary` icon tint, same `isLoading` trailing-spinner convention on the "Plan a route" parent card during GPS/settings resolution.

## No Analog Found

None — every file in scope has a direct, working in-repo analog (composition-only phase per RESEARCH.md's own framing; zero new infrastructure, zero new external packages).

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/util/`, `app/lib/provider/`, `app/lib/components/route_planner/`, `app/lib/models/`, `app/lib/entities/`
**Files scanned:** `trail_source_select_screen.dart`, `trail_import_util.dart`, `router_provider.dart`, `elevation_tab.dart`, `route_planner_screen.dart`, `gpx_util.dart` (referenced via RESEARCH.md, not re-read — already excerpted there), `settings.dart`/`settings_entity.dart` (referenced via RESEARCH.md), `web/src/lib/models/settings.ts` (cross-platform mirror source)
**Pattern extraction date:** 2026-07-17
