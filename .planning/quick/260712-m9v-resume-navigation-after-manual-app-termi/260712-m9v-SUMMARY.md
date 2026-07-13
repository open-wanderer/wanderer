---
phase: quick-260712-m9v
plan: 01
subsystem: mobile-navigation
tags: [flutter, objectbox, riverpod, freezed, gpx-navigation, resume-session]

# Dependency graph
requires:
  - phase: Phase 17 (Navigation on MapLibre)
    provides: navigation_screen.dart, navigation_provider.dart, navigation_stats_provider.dart, navigation_launch_util.dart
provides:
  - ActiveNavigationEntity (ObjectBox) persisting a single active nav/rec session row
  - active_navigation_store.dart best-effort save/read/clear helpers
  - Resume-seedable navigationProvider (progress + breadcrumb) and navigationStatsProvider (stats)
  - Periodic + event-driven persistence wired into NavigationScreen
  - Launch-time resume detection + AlertDialog in main.dart
affects: [future free-recording (rec) mode, any change to NavigationScreen/router_provider navigate route]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ActiveSessionType enum (nav|rec) persisted via @Transient()+int-getter/setter, mirroring TrailEntity's TrailDifficulty pattern"
    - "Single-row ObjectBox table pattern: save()/read()/clear() operate on 'the one active row' by construction, not by trailId lookup"
    - "Riverpod family resume-seed pattern: optional named params on build() threaded through every call site of the same family so it always resolves to the same provider instance"

key-files:
  created:
    - app/lib/entities/active_navigation_entity.dart
    - app/lib/util/active_navigation_store.dart
  modified:
    - app/lib/provider/navigation_provider.dart
    - app/lib/provider/navigation_stats_provider.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/provider/router_provider.dart
    - app/lib/util/navigation_launch_util.dart
    - app/lib/main.dart
    - app/lib/i18n/app_en.arb
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json

key-decisions:
  - "obxId is a settable constructor param on ActiveNavigationEntity (deviates from TrailEntity's style, which doesn't expose obxId in its constructor) — required so navigation_screen.dart can pass obxId: _activeRowObxId to update the tracked row on every save instead of inserting a duplicate."
  - "Localization keys in this codebase's AppLocalizations stay literal snake_case getters/methods (e.g. resume_navigation_prompt), not camelCase — confirmed against existing exit_navigation/stop_navigation_confirm before adding the new key."

patterns-established:
  - "Resume-seed threading: any future family provider needing app-restart resume support should follow navigation_provider.dart/navigation_stats_provider.dart's optional-named-param pattern so existing call sites never break."

requirements-completed: [QUICK-260712-m9v]

# Metrics
duration: ~20min
completed: 2026-07-12
---

# Quick Task 260712-m9v: Resume Navigation After Manual App Termination Summary

**Persisted in-progress turn-by-turn navigation (maneuver index, trip stats, polyline-encoded breadcrumb) to a single ObjectBox row so a manually-killed app relaunches into a resume dialog that reopens navigation exactly where it left off.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-12
- **Tasks:** 4/4
- **Files modified:** 26 (9 hand-written/entity files + objectbox codegen + riverpod/freezed codegen + 15 generated AppLocalizations locale files)

## Accomplishments

- New `ActiveNavigationEntity` (ObjectBox) with an `ActiveSessionType` enum (`nav`/`rec`) discriminator and nullable trail-specific fields, forward-compatible with a future trail-less recording mode without a schema migration.
- `active_navigation_store.dart`: best-effort, swallow-all `save`/`read`/`clear` helpers operating on "the single active row" — no `trailId` parameter needed.
- `navigationProvider` and `navigationStatsProvider` gained optional resume-seed parameters (`resumeManeuverIndex`/`resumeBreadcrumb`, and a new `NavigationStatsSeed` + `resume` param respectively) — every existing call site and unit test keeps compiling unchanged.
- `navigation_screen.dart` now writes an initial row on fresh navigation start, persists every 10s + immediately on maneuver advance + on pause toggle (always updating the same tracked row, including the polyline-encoded breadcrumb), and clears the row on deliberate exit.
- `router_provider.dart`'s `navigate` route now destructures a 3-tuple `(NavigateResponse, bool, ActiveNavigationEntity?)`, threading an optional resume seed from launch through to the screen.
- `main.dart`: `MainApp` converted to `ConsumerStatefulWidget`; a one-shot post-auth-settle check reads any persisted row, resolves its cached `NavigateResponse` via the now-public `readCachedNav`, and shows a resume `AlertDialog` naming the trail. Accept re-pushes navigation seeded from the row; decline (or an unresolvable/non-nav row) clears it silently.
- Added `resume_navigation_prompt` l10n key to `app_en.arb`; regenerated `AppLocalizations` via `flutter gen-l10n`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Persistence data layer (entity + store helper + codegen)** - `45b93ce4` (feat)
2. **Task 2: Make the two navigation providers resume-seedable** - `3b5fd30f` (feat)
3. **Task 3: Wire persistence into the screen + transport the resume seed** - `3784fc95` (feat)
4. **Task 4: Launch-time detection + resume dialog + localization** - `7503e481` (feat)

**Plan metadata:** committed separately by the orchestrator (docs commit — SUMMARY.md/STATE.md/ROADMAP.md), per this quick task's constraints.

## Files Created/Modified

- `app/lib/entities/active_navigation_entity.dart` - New ObjectBox entity; `ActiveSessionType` enum + nullable trail fields + shared breadcrumb/stats fields
- `app/lib/util/active_navigation_store.dart` - `save`/`read`/`clear` best-effort helpers over the single active row
- `app/lib/provider/navigation_provider.dart` - `build()` gains optional `resumeManeuverIndex`/`resumeBreadcrumb`, deriving `_currentShapeIndex` from the resumed maneuver's `beginShapeIndex`
- `app/lib/provider/navigation_stats_provider.dart` - New `NavigationStatsSeed` freezed class; `build()` gains optional `resume` param seeding `_start`/`_pausedAccum`/`_pauseStart`; new public `pausedAccum` getter
- `app/lib/routes/navigation_screen.dart` - `resumeSession` constructor param; `_persistNow()` writing to the tracked `obxId`; periodic timer + maneuver-advance + pause-toggle saves; clear-on-exit
- `app/lib/provider/router_provider.dart` - `navigate` route's `extra` guard/destructure widened to the 3-tuple; passes `resumeSession` to `NavigationScreen`
- `app/lib/util/navigation_launch_util.dart` - `_readCachedNav` → public `readCachedNav`; both push call sites updated to the 3-tuple shape (`null` resume seed for fresh launches)
- `app/lib/main.dart` - `MainApp` → `ConsumerStatefulWidget`; `_maybeResume()` launch-time detection + resume `AlertDialog`
- `app/lib/i18n/app_en.arb` (+ regenerated `app_localizations*.dart`) - New `resume_navigation_prompt` key with a `trail` placeholder
- `app/lib/objectbox.g.dart`, `app/lib/objectbox-model.json` - Regenerated to include `ActiveNavigationEntity`
- `app/lib/provider/navigation_provider.g.dart`, `app/lib/provider/navigation_stats_provider.g.dart`, `app/lib/provider/navigation_stats_provider.freezed.dart` - Regenerated for the new family params/seed type

## Decisions Made

- `ActiveNavigationEntity`'s constructor exposes `obxId` as a settable named param (not matching `TrailEntity`'s exact constructor style, which omits `obxId`) — required by Task 3's `_persistNow()`, which must pass `obxId: _activeRowObxId` so `active_nav.save()` updates the existing row rather than inserting a duplicate on every periodic/event-driven save.
- Confirmed this codebase's `AppLocalizations` getters/methods are literal snake_case (e.g. `exit_navigation`, `stop_navigation_confirm`) rather than the Flutter-default camelCase before adding `resume_navigation_prompt` — avoided a naming mismatch that would have failed at codegen/analyze time.

## Deviations from Plan

None — plan executed exactly as written across all 4 tasks. `flutter analyze`, `dart run build_runner build --delete-conflicting-outputs`, and `flutter gen-l10n` all ran as specified in each task's `<verify>` block with no new errors.

### Incidental, out-of-scope codegen drift (not committed)

Running `dart run build_runner build --delete-conflicting-outputs` in Task 1 also regenerated three unrelated riverpod-generated files whose source `.dart` files had been edited previously without a matching codegen run (stale hash drift, pre-existing before this session): `app/lib/provider/auth_provider.g.dart`, `app/lib/provider/trail/map_cluster_search_provider.g.dart`, `app/lib/provider/trail/trail_library_provider.g.dart`. Per the plan's scope boundary (only fix issues directly caused by this task's changes), these were left uncommitted and untouched in the working tree — they are harmless (just a doc-comment reformat + hash-string bump) and outside this plan's `files_modified` list.

---

**Total deviations:** 0 in-scope deviations. 1 out-of-scope, non-committed codegen side effect noted above for transparency.
**Impact on plan:** None — plan executed exactly as specified.

## Issues Encountered

- `flutter test` (whole suite) surfaced 3 pre-existing failures (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — these match the exact failures already documented as unrelated in STATE.md's Phase 18-01 deviation log (confirmed via git-stash bisect against the parent commit at that time). Not caused by this plan's changes; `flutter test test/provider/` (the plan's actual verification target) passes cleanly (24/24).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Feature is code-complete and passes all automated verification (build_runner codegen, gen-l10n, flutter analyze, flutter test test/provider/).
- **Manual on-device verification still required** (no device test harness available to the executor): start navigation, swipe-kill the app, relaunch, accept the resume dialog, and confirm maneuver index/distance/elevation/elapsed/breadcrumb all continue from the persisted values. Also verify (a) declining the dialog shows no prompt on next launch and (b) a deliberate Exit-then-relaunch shows no prompt. See the plan's `FOLLOW-UP` note for the full manual test script.

---
*Phase: quick-260712-m9v*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files verified present on disk; all 4 task commit hashes (45b93ce4, 3b5fd30f, 3784fc95, 7503e481) verified present in git log.
