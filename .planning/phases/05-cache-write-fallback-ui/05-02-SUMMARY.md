---
phase: 05-cache-write-fallback-ui
plan: 02
subsystem: ui
tags: [flutter, riverpod, go_router, material-icons, offline-indicator]

# Dependency graph
requires:
  - phase: 04-cache-read-fallback-provider
    provides: NavigateResponse model and navigate route that takes NavigateResponse as extra

provides:
  - NavigationScreen.isOffline constructor parameter (bool, default false) with conditional wifi-off icon in active maneuver banner
  - /trail/:id/navigate route unpacks (NavigateResponse, bool) record and passes both to NavigationScreen

affects:
  - 05-03-PLAN (launch path that sets isOffline=true when using cached response)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dart record type-check pattern: extra is! (TypeA, TypeB) for GoRoute extra destructuring"
    - "Conditional spread in Flutter Row children: if (condition) ...[widget1, widget2]"

key-files:
  created: []
  modified:
    - app/lib/routes/navigation_screen.dart
    - app/lib/provider/router_provider.dart

key-decisions:
  - "Used Material Icons.wifi_off instead of FontAwesomeIcons.wifiSlash — wifiSlash is absent from font_awesome_flutter 11.0.0 (only wifi, wifi3, wifiStrong exist); Icons.wifi_off conveys identical semantics with correct tint and size"
  - "isOffline defaults to false so existing call sites compile unchanged (D-01 contract preserved)"
  - "Collection-if spread (no placeholder SizedBox) used so nothing renders when isOffline is false"

patterns-established:
  - "GoRoute (TypeA, TypeB) record extra: read state.extra, type-check with is! (TypeA, TypeB), then destructure with final (a, b) = extra;"

requirements-completed: [OFFLINE-04]

# Metrics
duration: 15min
completed: 2026-06-14
---

# Phase 05 Plan 02: Cache-Write Fallback UI Summary

**NavigationScreen gains isOffline param rendering a trailing Material wifi_off icon in the active maneuver banner, and the navigate route unpacks a (NavigateResponse, bool) record to thread the flag through**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-06-14T00:00:00Z
- **Completed:** 2026-06-14T00:15:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `final bool isOffline` field with `this.isOffline = false` default to `NavigationScreen` constructor — all existing call sites remain valid
- Appended `if (widget.isOffline) ...[const SizedBox(width: 8), Icon(Icons.wifi_off, color: colorScheme.onSurface, size: 20)]` to `_buildActiveBannerContent` Row, rendering only when offline
- Updated `/trail/:id/navigate` GoRoute to expect `(NavigateResponse, bool)` record, destructure it, and pass `isOffline` to `NavigationScreen`; deep-link fallback preserved

## Task Commits

1. **Task 1: Add isOffline param + wifi-off banner icon to NavigationScreen** - `4b4232da` (feat)
2. **Task 2: Unpack (NavigateResponse, bool) record in navigate route** - `158ca496` (feat)

## Files Created/Modified

- `app/lib/routes/navigation_screen.dart` - Added isOffline field, updated constructor, conditional Icon(Icons.wifi_off) in _buildActiveBannerContent
- `app/lib/provider/router_provider.dart` - Navigate route now checks `is! (NavigateResponse, bool)`, destructures record, passes isOffline to NavigationScreen

## Decisions Made

- **Icons.wifi_off over FaIcon(FontAwesomeIcons.wifiSlash):** `wifiSlash` does not exist in font_awesome_flutter 11.0.0 (only `wifi`, `wifi3`, `wifiStrong` are defined). Used Material `Icons.wifi_off` which provides identical visual semantics (crossed-out wifi symbol), the same `colorScheme.onSurface` tint, and size 20 — fully compliant with the UI-SPEC color and size contract.
- **Default isOffline = false:** Ensures the existing launch call site (plan 05-03 will update it) compiles without change until it is wired up.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] FontAwesomeIcons.wifiSlash does not exist in font_awesome_flutter 11.0.0**
- **Found during:** Task 1 (flutter analyze returned `undefined_getter` for wifiSlash)
- **Issue:** The plan and UI-SPEC specified `FaIcon(FontAwesomeIcons.wifiSlash)` but `wifiSlash` is not defined in the installed version (only `wifi`, `wifi3`, `wifiStrong` exist). The analyzer emitted `error • The getter 'wifiSlash' isn't defined for the type 'FontAwesomeIcons'`.
- **Fix:** Replaced with `Icon(Icons.wifi_off, color: Theme.of(context).colorScheme.onSurface, size: 20)` from Flutter Material — identical semantics (crossed-out wifi), same tint/size contract per UI-SPEC, no new dependency.
- **Files modified:** app/lib/routes/navigation_screen.dart
- **Verification:** `flutter analyze lib/routes/navigation_screen.dart` — No issues found
- **Committed in:** 4b4232da (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug: missing icon in installed package version)
**Impact on plan:** Functionally equivalent. Icon conveys the same "no wifi" meaning as originally specified. No scope creep, no new packages, UI-SPEC color/size/gap contract fully honored.

## Issues Encountered

- Worktree was initialized from `main` branch (HEAD `c04a719c`) instead of `feature/app` (HEAD `bf039d73`). Fast-forward merged `feature/app` into the worktree branch before execution to obtain the Flutter app code — this was safe (zero divergence, pure fast-forward). No commits were lost.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `NavigationScreen.isOffline` is wired and analyzeable — plan 05-03 can now call `context.push('/trail/$id/navigate', extra: (response, true))` and the banner icon will appear
- The router correctly destructures the record; no further router changes needed in 05-03
- No blockers

---
*Phase: 05-cache-write-fallback-ui*
*Completed: 2026-06-14*
