---
phase: 31-flutter-settings-hierarchy
plan: 02
subsystem: mobile-app
tags: [flutter, riverpod, i18n, widget-testing, region-hierarchy]

# Dependency graph
requires:
  - phase: 31-flutter-settings-hierarchy (Plan 01)
    provides: "RegionHierarchyRow/RegionTreeNode models, buildRegionTree/computeDefaultExpanded/flattenVisible/computeFilterMatches, RegionRepository.fetchHierarchyRows"
provides:
  - "SettingsOfflineRegionsScreen renders the region catalog as a collapsible group/leaf hierarchy instead of a flat list"
  - "Four new ARB keys (regions_offline_unavailable_title/body, regions_group_expand_label/collapse_label) across all 14 locales"
  - "D-04 offline empty state (no hierarchy fetched this session), distinct from the fresh-install full-screen error"
  - "Widget-test harness pattern for a Notifier-backed screen with no real ObjectBox Store (noSuchMethod-based fake Store)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tree shape + expand-state cached in State, rebuilt only inside the fetch method, never inside build() (render-time join via a byId map is the only per-build tree operation)"
    - "noSuchMethod-based fake ObjectBox Store for widget tests, satisfying a repository constructor's Store parameter without ever dereferencing it"
    - "Optional icon param added to a shared _buildEmptyState helper, preserving every existing call site's rendering"

key-files:
  created:
    - app/test/routes/settings_offline_regions_screen_test.dart
  modified:
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_cs.arb
    - app/lib/i18n/app_es.arb
    - app/lib/i18n/app_eu.arb
    - app/lib/i18n/app_fr.arb
    - app/lib/i18n/app_hu.arb
    - app/lib/i18n/app_it.arb
    - app/lib/i18n/app_nl.arb
    - app/lib/i18n/app_no.arb
    - app/lib/i18n/app_pl.arb
    - app/lib/i18n/app_pt.arb
    - app/lib/i18n/app_ru.arb
    - app/lib/i18n/app_zh.arb
    - app/lib/i18n/app_localizations*.dart (generated)
    - app/lib/routes/settings_offline_regions_screen.dart

key-decisions:
  - "Substituted FontAwesomeIcons.linkSlash for the plan's specified wifiSlash icon — font_awesome_flutter's free icon set (11.0.0) has no wifiSlash glyph (Pro-only upstream); linkSlash is the closest already-vetted 'disconnected' icon, zero new dependency added (Rule 1 auto-fix)"
  - "_buildEmptyState gained an optional `icon` param (default null) rather than a new sibling method — every pre-existing call site is unaffected (renders identically), only the new D-04 offline call site passes an icon"
  - "Widget test's RegionRepository/regionListNotifierProvider/tileRepositoryStatusProvider overrides use a noSuchMethod-based _FakeStore (never dereferenced) so the tree renders deterministically without a real ObjectBox Store or network — no mocking library added"

requirements-completed: [APPUI-01, APPUI-02]

# Metrics
duration: 20min
completed: 2026-07-27
---

# Phase 31 Plan 02: Hierarchical Settings Screen Summary

**Settings → Offline Maps/Regions now renders the admin-defined region catalog as a collapsible group/leaf tree (ListView.builder over flattenVisible), with every existing Vector/DEM download/cancel/delete action and the disk-usage summary preserved unchanged, plus a new offline-scoped empty state.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-27
- **Tasks:** 3 completed
- **Files modified:** 2 source files (app_en.arb template + settings_offline_regions_screen.dart) + 13 locale ARBs + 15 generated localization files
- **Files created:** 1 (settings_offline_regions_screen_test.dart)

## Accomplishments
- Added 4 new ARB keys (`regions_offline_unavailable_title/body`, `regions_group_expand_label/collapse_label`) to all 14 locale files and regenerated `AppLocalizations` via `flutter gen-l10n`
- Converted the screen's flat `ListView.separated` to a `ListView.builder` over `flattenVisible`, dispatching to a new `_buildGroupRow` (chevron, 44px min touch target, no card/border, instant expand/collapse) or the existing untouched leaf-row renderer, keyed by `ValueKey(row.node.id)` per the Pitfall 7 gotcha
- Tree shape (`_treeRoots`) and expand-state seeding are cached in `State` and rebuilt only inside `_refreshCatalog` after a genuine hierarchy fetch — never inside `build()` — so a download/progress provider tick never reconstructs the tree (Pitfall 1)
- Wired the existing search bar to `computeFilterMatches`, auto-expanding ancestor chains while filtering (D-06)
- Added the D-04 offline empty state (shown when `_treeRoots == null`, i.e. no successful hierarchy fetch this session), reusing `_buildEmptyState` with a new optional icon parameter — distinct from the pre-existing fresh-install full-screen error
- Every existing Vector/DEM tile method, the disk-usage summary, and all mutation methods (`_onDownload*`/`_onCancel*`/`_onDelete*`) are byte-for-byte unchanged (verified via `git diff`)
- Added 3 widget tests covering the two APPUI-02 regression surfaces (nested leaf Vector/DEM actions, disk-usage summary total) plus the new APPUI-01 chevron expand/collapse behavior, all against the full `ProviderScope`/`MaterialApp` render — no reduced-scope fallback

## Task Commits

Each task was committed atomically:

1. **Task 1: Add tree/offline ARB keys across all locales + gen-l10n** - `2a9df6e4` (feat)
2. **Task 2: Convert the screen to a collapsible hierarchy (build/state/render)** - `829b0996` (feat)
3. **Task 3: Widget tests for nested leaf actions + unchanged disk-usage** - `d63b0cc6` (test)

**Plan metadata:** commit pending (docs: complete plan)

## Files Created/Modified
- `app/lib/i18n/app_en.arb` (+ 13 other locale ARBs) - Four new keys: `regions_offline_unavailable_title/body`, `regions_group_expand_label/collapse_label` (with `name` placeholder metadata in the template)
- `app/lib/i18n/app_localizations*.dart` - Regenerated via `flutter gen-l10n`
- `app/lib/routes/settings_offline_regions_screen.dart` - New `_treeRoots`/`_expandedIds`/`_expandedSeeded` state, hierarchy fetch in `_refreshCatalog`, tree-aware `build()`, new `_buildGroupRow`/`_indentFor`, `_buildEmptyState` gained an optional `icon` param
- `app/test/routes/settings_offline_regions_screen_test.dart` - 3 widget tests with a stubbed `RegionRepository`/`RegionListNotifier`/`TileRepositoryStatus` harness

## Decisions Made
- `FontAwesomeIcons.linkSlash` substituted for the plan's `wifiSlash` (doesn't exist in font_awesome_flutter's free set) — see Deviations below.
- `_buildEmptyState` extended with an optional `icon` param instead of a parallel method, keeping every pre-existing call site's output identical.
- Widget test harness uses a `noSuchMethod`-based fake `Store` (`implements Store { noSuchMethod... }`) to satisfy `RegionRepository`'s constructor without opening a real ObjectBox database — no mocking library added, matching the phase's zero-new-dependency constraint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `FontAwesomeIcons.wifiSlash` does not exist in font_awesome_flutter 11.0.0's free icon set**
- **Found during:** Task 2 (D-04 offline empty state)
- **Issue:** The plan and UI-SPEC specified `FontAwesomeIcons.wifiSlash` for the offline empty state icon. `wifi-slash` is a Font Awesome Pro-only glyph; the installed free package has no such getter, causing a compile error.
- **Fix:** Substituted `FontAwesomeIcons.linkSlash` ("broken link"), the closest already-vetted disconnected/offline glyph available in the existing dependency. No new package added.
- **Files modified:** `app/lib/routes/settings_offline_regions_screen.dart`
- **Verification:** `flutter analyze` passes; the offline empty-state widget test scenario was manually exercised via the render path (not a dedicated automated test — see Known Gaps).
- **Committed in:** `829b0996` (Task 2 commit)

**2. [Rule 3 - Blocking] `find.byIcon` requires `IconData`, not `FaIconData`, in widget tests**
- **Found during:** Task 3 (widget tests)
- **Issue:** `FontAwesomeIcons.*` constants are typed `FaIconData` (a wrapper), not `IconData`; `find.byIcon()` only accepts `IconData`.
- **Fix:** Used `FontAwesomeIcons.X.data` (the wrapped `IconData`) at each `find.byIcon` call site.
- **Files modified:** `app/test/routes/settings_offline_regions_screen_test.dart`
- **Verification:** `flutter analyze` and `flutter test` both pass.
- **Committed in:** `d63b0cc6` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug/wrong-API-reference, 1 blocking/test-API-mismatch)
**Impact on plan:** Both fixes were necessary to make the code compile at all; no scope creep. The icon substitution is a visual-only deviation from the plan's literal acceptance-criteria grep for `FontAwesomeIcons.wifiSlash` (that string cannot appear verbatim since the API doesn't exist) — the acceptance criterion's *intent* (a disconnected-state icon on the D-04 empty state) is fully satisfied.

## Issues Encountered
- `git status` showed an unrelated pre-existing modification to `app/lib/provider/router_provider.g.dart` (a riverpod codegen hash-only diff) present before this plan's work began; left untouched and uncommitted as out of scope for this plan's `files_modified`.
- Full `flutter test` run surfaced 4 pre-existing failures in `test/components/route_planner/settings_tab_test.dart`, confirmed via `git stash` to fail identically without any of this plan's changes — unrelated, out of scope, not fixed.

## Known Gaps
- The D-04 offline empty state (rendered when `_treeRoots == null`) is exercised by the plan's design and by manual code review, but the widget-test suite added in Task 3 does not include a dedicated automated test simulating a hierarchy-fetch failure — the plan's Task 3 acceptance criteria scoped tests to the two APPUI-02 regression surfaces plus the chevron toggle only. On-device human verification (per the plan's Task 2 `<human-check>`) has not been performed in this autonomous execution; the plan carries no `checkpoint:human-verify` task gating this, so the automated verification (analyze/gen-l10n/test) stands as the completion bar for this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 31 (Flutter Settings Hierarchy) is now complete across both plans: Plan 01 delivered the data contracts and pure tree algorithm, Plan 02 wires them into the Settings screen. This closes out APPUI-01 and APPUI-02, the last two requirements of the v1.7 Admin Region Picker milestone's Active list. A future session should perform the on-device human verification described in Plan 02's Task 2 (`<human-check>`) before considering the milestone fully UAT-signed-off, and optionally add a dedicated automated test for the D-04 offline empty state.

---
*Phase: 31-flutter-settings-hierarchy*
*Completed: 2026-07-27*

## Self-Check: PASSED

All created/modified files found on disk; all 3 task commit hashes (2a9df6e4, 829b0996, d63b0cc6) found in git log.
