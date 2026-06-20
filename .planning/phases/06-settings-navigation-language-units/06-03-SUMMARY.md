---
phase: 06-settings-navigation-language-units
plan: 03
subsystem: mobile-app
tags: [riverpod, i18n, units, format, elevation-profile]

# Dependency graph
requires:
  - phase: 06-01
    provides: unitProvider (derived Riverpod provider reading settings.unit)
provides:
  - Every format_util call site (14 files, ~50 calls) reads the live unit from unitProvider
  - elevation_profile.dart converted to ConsumerStatefulWidget (unit-aware)
  - waypoint_sheet.dart converted to ConsumerWidget (unit-aware)
  - trail_timeline.dart converted to ConsumerWidget threading unit to _TimelineRow
  - imperial conversion tests for formatDistance + formatElevation
affects:
  - Phase 7-9 settings screens (units toggle now visibly re-renders all stats)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single ref.watch(unitProvider) per build path, threaded to private stat widgets/helpers via constructor or method param"
    - "ConsumerStatefulWidget instance field (_unit) assigned at top of build so fl_chart axis closures see live unit during the same build"
    - "Constructor-param threading keeps nested private widgets (_TimelineRow) ref-free"

key-files:
  created: []
  modified:
    - app/lib/components/trail/trail_card.dart
    - app/lib/components/trail/trail_list_item.dart
    - app/lib/components/trail/trail_panel.dart
    - app/lib/components/trail/trail_quick_filter_bar.dart
    - app/lib/components/trail/summit_log_card.dart
    - app/lib/components/list/list_card.dart
    - app/lib/components/list/list_list_item.dart
    - app/lib/routes/global_search_screen.dart
    - app/lib/routes/list_detail_screen.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/routes/trail_filter_screen.dart
    - app/lib/components/trail/elevation_profile.dart
    - app/lib/components/trail/waypoint_sheet.dart
    - app/lib/components/trail/trail_timeline.dart
    - app/test/util/format_util_test.dart

key-decisions:
  - "Used a single ref.watch(unitProvider) per build path and threaded the value down rather than re-reading the provider in nested helpers/widgets"
  - "elevation_profile.dart stores unit in a _unit instance field assigned at the top of build so the fl_chart getTitlesWidget closures (which run during the same build) read the live value"
  - "trail_timeline TrailTimeline became a ConsumerWidget while _TimelineRow stays a plain StatelessWidget receiving unit via constructor (ref-free private widget)"

patterns-established:
  - "Unit-aware display: every format_util call passes unit: from unitProvider; no implicit metric default remains"

requirements-completed: [LANG-02]

# Metrics
duration: ~22min
completed: 2026-06-19
---

# Phase 6 Plan 03: Unit-Aware Display Wiring Summary

**Every format_util call site (14 files, ~50 calls) now reads the live metric/imperial preference from unitProvider, so toggling units in settings re-renders distances, elevations, and speeds across trail cards, lists, navigation, and the elevation profile — backed by new imperial conversion tests.**

## Performance

- **Duration:** ~22 min
- **Started:** 2026-06-19T23:05:18Z (after 06-02 metadata commit)
- **Completed:** 2026-06-19T23:23:54Z
- **Tasks:** 2
- **Files modified:** 15

## Accomplishments
- Wired `unitProvider` into all 11 Consumer call-site files, threading `unit` to private stats widgets (`_StatsGrid`, `_StatsRow`, `_FullContent`, `_TrailTile`, `_ResultsList`) and navigation banner/stats helpers
- Converted the 3 non-Consumer widgets: `elevation_profile.dart` → `ConsumerStatefulWidget`, `waypoint_sheet.dart` → `ConsumerWidget`, `trail_timeline.dart` → `ConsumerWidget` with constructor-threaded `_TimelineRow`
- Added imperial coverage to `format_util_test.dart`: `formatDistance(1000, unit: 'imperial')` → `0.62 mi`, `formatElevation(100, unit: 'imperial')` → `328 ft`, plus metric-default assertions
- Eliminated the implicit metric default at every call site (verified repo-wide with a paren-matching scan)

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire unitProvider into the 11 Consumer call sites** - `83598d17` (feat)
2. **Task 2: Convert the 3 non-Consumer call sites and add imperial format tests** - `5cf6684b` (feat)

## Files Created/Modified
- `app/lib/components/trail/trail_card.dart` - `unit` watched in build, threaded to `_StatsGrid`
- `app/lib/components/trail/trail_list_item.dart` - threaded to `_StatsRow`
- `app/lib/components/trail/trail_panel.dart` - 3 direct format calls pass `unit`
- `app/lib/components/trail/trail_quick_filter_bar.dart` - 4 calls in the elevation bottom-sheet Consumer closure pass `unit`
- `app/lib/components/trail/summit_log_card.dart` - 3 calls pass `unit`
- `app/lib/components/list/list_card.dart` - threaded to `_FullContent`
- `app/lib/components/list/list_list_item.dart` - threaded to `_StatsRow`
- `app/lib/routes/global_search_screen.dart` - threaded through `_ResultsList` → `_TrailTile`
- `app/lib/routes/list_detail_screen.dart` - `_ListHeader` Consumer reads `unit`
- `app/lib/routes/navigation_screen.dart` - `unit` watched in build, threaded through `_buildBanner` → `_buildActiveBannerContent` and `_buildStatsSheet` → `_buildAdditionalStats` (6 calls)
- `app/lib/routes/trail_filter_screen.dart` - 6 calls in state build pass `unit`
- `app/lib/components/trail/elevation_profile.dart` - `ConsumerStatefulWidget`, `_unit` field assigned in build, 7 calls (incl. fl_chart axis closures) use it
- `app/lib/components/trail/waypoint_sheet.dart` - `ConsumerWidget`, single call uses `unit`
- `app/lib/components/trail/trail_timeline.dart` - `ConsumerWidget`; `_TimelineRow` gains a `unit` constructor param used at both calls
- `app/test/util/format_util_test.dart` - added `formatDistance`/`formatElevation` groups with imperial + metric cases

## Decisions Made
- Single `ref.watch(unitProvider)` per build path, threaded down via constructor/method params rather than re-reading the provider in nested helpers (per plan PATTERNS guidance).
- `elevation_profile.dart` uses a `_unit` instance field set at the top of `build` so the fl_chart `getTitlesWidget` closures see the live unit during the same build frame.
- `trail_timeline.dart`'s private `_TimelineRow` stays ref-free, receiving `unit` by constructor; the public `TrailTimeline` reads the provider.

## Deviations from Plan

None - plan executed exactly as written.

Note: while editing `trail_timeline.dart` I dropped the never-supplied `key` parameter from the private `_TimelineRow` constructor (a pre-existing `unused_element_parameter` analyzer warning) because the constructor was already being modified to add the `unit` param. This was incidental cleanup within a file already being changed, not separate scope.

## Issues Encountered
- `trail_filter_screen.dart` line 230 uses `formatElevation` for a distance-slider max label (a pre-existing label inconsistency). Left as-is per scope boundary — only `unit:` was added to preserve existing behavior; not a bug introduced by this plan.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- LANG-02 display half (D-04/D-05) is complete: toggling imperial/metric in the Language & Units screen now re-renders all distance/elevation/speed displays app-wide.
- No blockers. Plan 04 (final Phase 6 plan) can proceed.

## Known Stubs
None.

## Self-Check: PASSED

- FOUND: app/lib/components/trail/elevation_profile.dart (ConsumerStatefulWidget, unitProvider)
- FOUND: app/lib/components/trail/waypoint_sheet.dart (ConsumerWidget, unitProvider)
- FOUND: app/lib/components/trail/trail_timeline.dart (_TimelineRow unit param)
- FOUND: app/test/util/format_util_test.dart (imperial cases: 0.62 mi, 328 ft)
- FOUND: commit 83598d17
- FOUND: commit 5cf6684b
- VERIFIED: repo-wide paren-matching scan — every format_util call site passes unit:
- VERIFIED: flutter analyze on all 14 call-site files — no new errors
- VERIFIED: flutter test test/util/format_util_test.dart — 14 passed

---
*Phase: 06-settings-navigation-language-units*
*Completed: 2026-06-19*
