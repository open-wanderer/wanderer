---
phase: 21-route-planner-handoff-entry-point
plan: 03
subsystem: mobile-route-planner-entry-point
tags: [flutter, go_router, riverpod, geolocator, route-planner]

# Dependency graph
requires:
  - phase: 21-route-planner-handoff-entry-point
    plan: 02
    provides: "Settings.behavior?.allowAutoGeolocate field, gating GPS resolution at planner entry"
provides:
  - "showTravelProfileSheet(context) — dismissible hike/bike modal bottom sheet returning 'pedestrian'/'bicycle'/null"
  - "TrailSourceSelectScreen's 'Plan a route' card → sheet → GPS-gated-or-fallback initialCenter → real /route-planner push"
  - "/route-planner real route registration reading travelProfile/lat/lon from extra (TEMPORARY Phase-19 stub removed)"
affects: [21-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "showModalBottomSheet with the app-wide 20px top-corner-radius sheet chrome (isDismissible: true, enableDrag: true) reused for a two-card selection sheet"
    - "GPS-gated initial-center resolution: Settings.behavior?.allowAutoGeolocate gate -> one-shot firstWhere((p) => p != null).timeout(4s) on foregroundPositionStreamProvider -> mapCameraProvider -> settings.location -> (0,0) fallback chain"

key-files:
  created:
    - app/lib/components/route_planner/travel_profile_sheet.dart
  modified:
    - app/lib/routes/trail_source_select_screen.dart
    - app/lib/provider/router_provider.dart
    - app/lib/i18n/app_en.arb

key-decisions:
  - "_TravelProfileCard replicates _SourceActionCard's shape locally rather than importing the private widget, per the plan's explicit self-containment instruction"
  - "_resolveInitialCenter/_fallbackCenter fallback chain: mapCameraProvider (last saved camera) -> settings.location (saved home) -> Geographic(0,0) — simplest existing precedents per D-03's implementer's-discretion note, never silently defaulting to GPS when allowAutoGeolocate isn't true"
  - "Reused the existing _importing StatefulWidget flag (not a new field) to drive the 'Plan a route' card's isLoading spinner during center resolution, matching the Import card's established idiom"

patterns-established:
  - "Any future entry-point selection sheet (two or more _SourceActionCard-style options) follows travel_profile_sheet.dart's shape: showModalBottomSheet<T> with the 20px sheet-chrome constant, a centered drag-handle bar, and Navigator.pop(context, value) per card — no forced Continue/Cancel step"

requirements-completed: [HANDOFF-02, HANDOFF-03]

# Metrics
duration: 12min
completed: 2026-07-17
---

# Phase 21 Plan 03: Route Planner Entry Point — Hike/Bike Sheet & Real /route-planner Wiring Summary

**Real HANDOFF-02/03 entry point: the "Plan a route" card now opens a dismissible hike/bike bottom sheet, resolves a GPS-gated (or fallback) initial map center via `Settings.behavior?.allowAutoGeolocate`, and pushes `/route-planner` with the chosen travel profile — replacing both TEMPORARY Phase-19 stubs (the hardcoded route registration and the card's direct push).**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-17T14:58:14Z
- **Completed:** 2026-07-17T15:10:00Z
- **Tasks:** 2
- **Files modified:** 3 hand-edited (`travel_profile_sheet.dart` new, `trail_source_select_screen.dart`, `router_provider.dart`) + `app_en.arb` + 15 regenerated `AppLocalizations` files

## Accomplishments

- Created `travel_profile_sheet.dart`: `showTravelProfileSheet(context)` presents a dismissible `showModalBottomSheet` (the app-wide 20px sheet chrome, `isDismissible: true`, `enableDrag: true`) with a centered drag-handle bar and two `_TravelProfileCard`s (Hike/`personHiking`, Bike/`bicycle`) — each card both selects and closes the sheet (`Navigator.pop(context, 'pedestrian'|'bicycle')`), no forced-choice Continue/Cancel step (D-01/D-02)
- `_TravelProfileCard` mirrors `_SourceActionCard`'s exact visual shape (16px card radius, 12px icon-badge radius, `secondaryContainer`@40%-alpha badge, `theme.colorScheme.primary` icon tint) — self-contained, no import of the private widget from `trail_source_select_screen.dart`
- Added `travel_profile_hike(_description)`, `travel_profile_bike(_description)`, and `finish_disabled_hint` (consumed by Plan 04) l10n keys to `app_en.arb`; regenerated `AppLocalizations`
- Wired `TrailSourceSelectScreen`: `_openPlanner` opens the sheet, resolves `initialCenter` via `_resolveInitialCenter` (GPS-gated on `Settings.behavior?.allowAutoGeolocate`, one-shot fix with a 4s timeout, falling back to `_fallbackCenter`'s `mapCameraProvider` → `settings.location` → `(0,0)` chain), then pushes `/route-planner` with `{'travelProfile', 'lat', 'lon'}`; reused the existing `_importing` flag so the "Plan a route" card shows the same loading spinner the Import card already uses
- Deleted both TEMPORARY Phase-19 markers: the card's direct `context.push('/route-planner')` and `router_provider.dart`'s hardcoded `/route-planner` registration, replaced with a real registration reading `travelProfile`/`lat`/`lon` off `state.extra` (mirroring the existing `/map` route's `Map<String, dynamic>` extra pattern), null-safe-falling-back to `'pedestrian'`/`Geographic(0,0)` if extra is ever dropped
- `flutter analyze` on all three touched files (plus whole-app) reports no new issues (46 pre-existing info/warning-level issues unrelated to this plan's files)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create travel_profile_sheet.dart — dismissible hike/bike modal sheet (D-01/D-02, HANDOFF-03)** - `8ce97d2d` (feat)
2. **Task 2: Wire the entry flow — card onTap → sheet → GPS-gated center → push /route-planner; replace TEMPORARY router registration (HANDOFF-02/03, D-03)** - `3163055b` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/components/route_planner/travel_profile_sheet.dart` - NEW: `showTravelProfileSheet` + `_TravelProfileCard`
- `app/lib/routes/trail_source_select_screen.dart` - "Plan a route" card `onTap` → `_openPlanner`; new `_openPlanner`/`_resolveInitialCenter`/`_fallbackCenter` methods
- `app/lib/provider/router_provider.dart` - `/route-planner` real registration reading `travelProfile`/`lat`/`lon` from `extra`
- `app/lib/i18n/app_en.arb` + regenerated `AppLocalizations*.dart` - 5 new l10n keys

## Decisions Made

- `_TravelProfileCard` replicates `_SourceActionCard`'s shape locally (not imported) — keeps the new component self-contained per the plan's explicit instruction
- Fallback chain for `_fallbackCenter`: last saved `mapCameraProvider` position, then `settings.location` (saved home), then `Geographic(lat: 0, lon: 0)` as the last resort — simplest existing precedents per D-03's implementer's-discretion note
- Reused the existing `_importing` `StatefulWidget` flag (not a new field) to drive the "Plan a route" card's `isLoading` spinner during center resolution — this also means both cards briefly disable together while the sheet's GPS/fallback resolution runs, matching the `isLoading` convention the Import card already established

## Deviations from Plan

None - plan executed exactly as written; both files edited per the plan's exact `<action>` spec, all acceptance-criteria greps pass verbatim.

## Issues Encountered

None. `flutter gen-l10n` ran cleanly (only pre-existing untranslated-message warnings for non-English locales, unrelated to this plan's new keys — English is the hard requirement per RESEARCH.md).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `travel_profile_sheet.dart`, the real `/route-planner` registration, and the GPS-gated `initialCenter` resolution are all in place for Plan 04's app-bar Finish action / handoff sequence wiring
- `finish_disabled_hint` l10n key already added in this plan's Task 1 (per the plan's single-arb-owner instruction) — ready for Plan 04 to consume
- No blockers for Plan 04

---
*Phase: 21-route-planner-handoff-entry-point*
*Completed: 2026-07-17*

## Self-Check: PASSED

- FOUND: app/lib/components/route_planner/travel_profile_sheet.dart
- FOUND: app/lib/routes/trail_source_select_screen.dart
- FOUND: app/lib/provider/router_provider.dart
- FOUND commit: 8ce97d2d
- FOUND commit: 3163055b
