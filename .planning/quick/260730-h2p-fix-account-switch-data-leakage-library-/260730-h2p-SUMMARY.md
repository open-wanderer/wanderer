---
phase: quick-260730-h2p
plan: 01
subsystem: mobile-app-auth
tags: [flutter, riverpod, objectbox, account-switching, privacy, offline-regions]

# Dependency graph
requires: []
provides:
  - "Account-scoped local data purge (ObjectBox rows + on-disk dirs) wired into logout and detected account switches"
  - "Declarative keepAlive-provider invalidation list, invoked on any auth user-id change"
  - "OwnProfile watching auth so it rebuilds on account switch instead of caching the previous account's Actor"
  - "Startup self-heal reconciling RegionEntity package status/links against archives actually on disk, plus an orphan region-directory sweep"
affects: [mobile-app-region-downloads, mobile-app-profile, mobile-app-library]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Account-change chokepoint purge: logout() and _updateUserEntity() both call purgeAccountScopedData, gated by a pure shouldPurgeForIncomingUser predicate"
    - "Declarative accountScopedProviders list + single invalidation loop reused from both widget-tree code (ref.invalidate) and a bare ProviderContainer (container.invalidate) for testability"
    - "DB-to-disk self-heal: pure resolvePackageRepair/orphanedRegionDirNames truth tables, Store-driven pass wraps its body in try/catch and never throws so a startup fault can't block app launch"

key-files:
  created:
    - app/lib/util/account_data_purge_util.dart
    - app/lib/util/account_scope_invalidation.dart
    - app/lib/util/region_disk_reconcile_util.dart
    - app/test/util/account_data_purge_util_test.dart
    - app/test/util/account_scope_invalidation_test.dart
    - app/test/util/region_disk_reconcile_util_test.dart
  modified:
    - app/lib/provider/auth_provider.dart
    - app/lib/provider/profile/profile_provider.dart
    - app/lib/main.dart

key-decisions:
  - "Chose clear-on-switch (deletion) over scope-by-user-id filtering for the privacy fix — filtering leaves files on disk readable outside the app's query layer; deletion is structural and closes the leak regardless of read path"
  - "Region archives/DEM archives/glyph cache/device preferences deliberately excluded from the purge — expensive to re-download, not tied to any one account"
  - "Region download-state divergence gets a self-heal (reconcile against disk at startup) rather than a diagnosis-dependent patch, fixing both the disk-usage/status mismatch and the offline-map skip in one pass"

patterns-established:
  - "accountScopedDirNames allow-list (library, avatars) as the single source of truth for what an account-scoped purge deletes on disk"
  - "accountScopedProviders as the single source of truth for what an account switch invalidates in memory"

requirements-completed: [SWITCH-LIBRARY, SWITCH-REGIONS, SWITCH-PROFILE]

# Metrics
duration: ~35min
completed: 2026-07-30
---

# Quick Task 260730-h2p: Fix Account-Switch Data Leakage Summary

**Structural purge-on-switch fix for three account-switch defects: local library/waypoint/actor rows and files now deleted at logout and on detected account mismatch, every keepAlive provider holding account-scoped state is invalidated on auth change, own-profile watches auth instead of caching it, and a startup self-heal reconciles region download status against what's actually on disk.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3/3 completed
- **Files modified:** 9 (6 created, 3 modified)

## Accomplishments
- Closed the privacy leak where account B could see account A's downloaded trails/waypoints/actors/library files after a switch — both at logout and defensively on any detected account-id mismatch
- Every keepAlive Riverpod cache holding account-scoped server data (own profile, search/save/filter, categories, region list, tile repo status) is now dropped in one declarative sweep on auth change
- Own-profile screen now shows the newly signed-in account immediately, no manual pull-to-refresh needed
- A region whose archive is on disk but whose ObjectBox link was lost (or vice versa) now self-heals at every app startup, keeping the disk-usage summary, the region row status, and the offline map's tile source in agreement
- Unreachable region directories (matching no persisted row) are swept, gated so an empty local catalog can never trigger a mass-delete

## Task Commits

Each task was committed atomically:

1. **Task 1: Purge account-scoped local data on logout and on account switch** - `fc6712d1` (feat)
2. **Task 2: Drop account-scoped in-memory caches on auth change and make own-profile follow auth** - `b2b8bc82` (feat)
3. **Task 3: Self-heal region download state against the archives on disk at startup** - `4574dc5e` (feat)

_No plan-metadata commit — the executor's docs commit (SUMMARY.md/STATE.md) is made separately by the orchestrator per this quick task's constraints._

## Files Created/Modified
- `app/lib/util/account_data_purge_util.dart` - `shouldPurgeForIncomingUser` predicate, `purgeAccountScopedDirectories`, `purgeAccountScopedBoxes`, `purgeAccountScopedData` entry point
- `app/lib/util/account_scope_invalidation.dart` - `accountScopedProviders` list, `invalidateAccountScopedProviders`/`invalidateAccountScopedProvidersOn`
- `app/lib/util/region_disk_reconcile_util.dart` - `resolvePackageRepair`, `orphanedRegionDirNames`, `reconcileRegionPackagesWithDisk`
- `app/lib/provider/auth_provider.dart` - `logout()` purges after clearing `UserEntity`; `_updateUserEntity` defensively purges on a detected account-id mismatch before writing incoming data
- `app/lib/provider/profile/profile_provider.dart` - `OwnProfile.build()` watches `authProvider.future` instead of reading it once
- `app/lib/main.dart` - existing `_authSub` listener invalidates account-scoped providers on a user-id change; `reconcileRegionPackagesWithDisk` awaited between `openStore` and `TileProxyServer.start`
- `app/test/util/account_data_purge_util_test.dart` - temp-dir coverage for the predicate + on-disk purge
- `app/test/util/account_scope_invalidation_test.dart` - membership/no-duplicate/exclusion assertions on the provider list
- `app/test/util/region_disk_reconcile_util_test.dart` - full truth table for `resolvePackageRepair` and `orphanedRegionDirNames`

## Decisions Made
- Clear-on-switch (deletion), not scope-by-user-id filtering, per the plan's pre-resolved (a) vs (b) vs (c) analysis — filtering alone can't satisfy "no path may leave account A's trails readable by account B" since files on disk stay readable regardless of query filters
- `UserEntity` box is cleared by `Auth` at both its own call sites, not by `purgeAccountScopedBoxes` — keeps ownership of that box with the notifier that already manages it
- `ProviderOrFamily` (used for `accountScopedProviders`' element type) required importing `package:flutter_riverpod/misc.dart` explicitly — the main `flutter_riverpod.dart` barrel does not re-export it even though it's a real public type used by `WidgetRef.invalidate`'s signature

## Deviations from Plan

None - plan executed exactly as written. All file names, symbol names, exports, and wiring points matched the plan's `must_haves.artifacts`/`key_links` specification exactly.

## Issues Encountered
- `ProviderOrFamily` failed to resolve as a type when only `package:flutter_riverpod/flutter_riverpod.dart` was imported, despite the plan's context asserting it as "exported by `package:flutter_riverpod/flutter_riverpod.dart`". Traced through the package source: the symbol is genuinely defined and used internally (`widget_ref.dart`'s `invalidate` signature), but the main barrel only exports it via a separate `misc.dart` sub-library. Fixed by adding `import 'package:flutter_riverpod/misc.dart' show ProviderOrFamily;` alongside the main import — no scope change, just an additional import to resolve a real package-structure detail the plan's context got slightly wrong about which file exports it.
- `flutter test` (full suite) surfaces 4 pre-existing failures in `test/components/route_planner/settings_tab_test.dart` (icon-finder assertions unrelated to auth/profile/region code) not previously logged in Phase 18's `deferred-items.md`. Verified via a disposable `git worktree` checked out at the parent commit (863f3181, before any of this plan's changes) that all 4 failures reproduce identically there — confirmed pre-existing and unrelated to this plan's `files_modified`, not fixed here per the plan's scope boundary. Worktree was removed after verification; no lasting changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All three defects (library leak, region status/disk divergence, stale own-profile) are fixed with unit test coverage on every pure helper; `reconcileRegionPackagesWithDisk` and the ObjectBox-dependent purge/invalidation paths still need the on-device verification described in the plan's Task 3 `<human-check>` block (account-switch library/profile/region checks, plus a full app kill/relaunch) before this is considered field-verified.
- No blockers for other in-flight work — this task only touched auth/profile/region-reconcile wiring, no schema or API changes.

---
*Phase: quick-260730-h2p*
*Completed: 2026-07-30*

## Self-Check: PASSED

All 9 created/modified source and test files confirmed present on disk; all 3 task commits (`fc6712d1`, `b2b8bc82`, `4574dc5e`) confirmed in git history.
