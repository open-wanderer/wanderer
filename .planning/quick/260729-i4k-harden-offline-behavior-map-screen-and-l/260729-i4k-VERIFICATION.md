---
phase: quick-260729-i4k
verified: 2026-07-29T15:40:00Z
status: passed
score: 8/8 must-haves verified
overrides_applied: 0
---

# Quick Task 260729-i4k: Harden Offline Behavior Verification Report

**Task Goal:** Harden offline behavior: map screen and list screen show clear offline message with retry CTA instead of errors; bottom nav profile picture cached and shown offline; profile screen shows cached profile with settings always available instead of spinner-then-error, and settings network-mutating actions disabled/deferred while offline
**Verified:** 2026-07-29T15:40:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Opening the map screen while offline shows a static offline message with a Try again button instead of a raw exception string, and the trail bottom sheet stays present and draggable | VERIFIED | `map_screen.dart:236,363-372` watches `onlineStatusProvider` and swaps `TrailCollectionMap` for `WandererOfflineState` when offline; `ValueKey('trail_sheet')` at line 611 appears exactly once (untouched, later Stack child, still paints over the takeover) |
| 2 | Opening the list screen while offline shows the same offline message with a Try again button instead of the generic Something went wrong error | VERIFIED | `list_screen.dart:65,114-120` branches inside the `Expanded`/`RefreshIndicator` to `WandererOfflineState` instead of `AsyncLoader`→`WandererError` |
| 3 | Tapping Try again while still offline re-probes and stays on the offline state; tapping it after connectivity returns re-renders map tiles / list results without an app restart | VERIFIED | `map_screen.dart:218-231` `_retryOnline` calls `refresh()`, early-returns if still offline, else invalidates `mapStyleSourcesProvider`/`mapStyleJsonProvider` and re-runs bounds search; `list_screen.dart:56-60` `_retryOnline` calls `refresh()` then invalidates `listSearchProvider`. `WandererOfflineState`'s `_inFlight` guard (verified by widget test) prevents double-fire while probing |
| 4 | The bottom-nav profile avatar renders the user's picture while offline after any prior online session, with a user glyph fallback instead of a blank grey circle | VERIFIED | `wanderer_layout.dart` `_NavAvatar` (190-267): `FileImage` from `cachedAvatarFile` first, then `NetworkImage` while online, else `FaIcon(user)` glyph — no bare `CircleAvatar` with no fallback remains; discarded-return `onBackgroundImageError` bug fixed (now sets `_networkFailed` via `setState`) |
| 5 | Opening the profile screen while offline renders cached name and avatar immediately with no spinner-then-error, and the settings gear is tappable and opens SettingsScreen | VERIFIED | `profile_screen.dart:87-100` own-profile `loading`/`error` branches both render `_buildCachedProfile(cachedUser)` when a cached `UserEntity` exists; `_buildCachedProfile` (122-171) renders the settings gear `IconButton` pushing `/settings`, the cached avatar, and username. `context.push('/settings')` appears twice (normal path + cached-profile path) |
| 6 | An ordinary HTTP error response (404/422/500) never marks the app offline, because the server was reached | VERIFIED | `isConnectionFailure` (`online_status_provider.dart:28-35`) returns false for `badResponse`/`badCertificate`/`cancel`; test suite covers 404/422/500/503 explicitly, all pass |
| 7 | Settings and all its sub-screens stay browsable while offline — every screen still renders its rows and no screen replaces its body with an error | VERIFIED | `SettingsOfflineBanner` inserted as first body child in account/privacy/notifications/language/categories/subcategories screens (grep-confirmed); no screen swaps its body for an offline takeover — banner is a non-replacing inline strip |
| 8 | Attempting a network-dependent settings action while offline (avatar upload, delete account, bio save, email change, password change, privacy/notification/language toggles, category or subcategory toggle and reorder) is disabled or blocked with a clear offline indication, rather than silently failing or throwing a raw DioException | VERIFIED | `guardOnline` present as first statement in: `settings_account_screen.dart` (avatar upload, delete, bio save — 3 call sites), `email_change_sheet.dart`/`password_change_sheet.dart` `_submit()`, privacy/notifications/language `_save`, categories/subcategories `_save` and `_onReorder` (with `_reordering` flag correctly cleared on the guard's early-return path, mirroring the catch path) |

**Score:** 8/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/provider/online_status_provider.dart` | keepAlive OnlineStatus notifier + isConnectionFailure classifier | VERIFIED | `class OnlineStatus` present, keepAlive, `build()` returns `true` and reads no provider; exports `OnlineStatus`, `onlineStatusProvider` (via codegen), `isConnectionFailure`; 69 lines |
| `app/lib/components/base/wanderer_offline_state.dart` | Shared offline empty state with retry CTA | VERIFIED | `class WandererOfflineState`, stateful, `_inFlight` guard, plain-string-only params; 91 lines |
| `app/lib/util/avatar_cache_util.dart` | Deterministic on-disk avatar cache, path-traversal safe | VERIFIED | `cachedAvatarFileName` uses `p.basename(Uri.parse(...).path)`; 86 lines |
| `app/lib/util/connectivity_util.dart` | Dio-based reachability probe (no WidgetRef) | VERIFIED | `Future<bool> isBackendReachable(Dio api)` — no `WidgetRef`, no riverpod import |
| `app/lib/provider/api_provider.dart` | Interceptor feeding OnlineStatus from every shared-client request | VERIFIED | `InterceptorsWrapper` added after `CookieManager`, resolves notifier lazily inside closures |
| `app/lib/util/offline_guard_util.dart` | Shared pre-flight gate blocking network mutation while offline | VERIFIED | `bool guardOnline(WidgetRef ref, AppLocalizations l10n)`, synchronous, `ref.read`-based; 41 lines |
| `app/lib/components/settings/settings_offline_banner.dart` | Single shared offline banner for settings sub-screens | VERIFIED | `class SettingsOfflineBanner extends ConsumerWidget`; 49 lines |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `api_provider.dart` | `onlineStatusProvider.notifier` | `InterceptorsWrapper` onResponse/onError | WIRED | `ref.read(onlineStatusProvider.notifier)` resolved inside both closures |
| `map_screen.dart` | `mapStyleSourcesProvider` | Try again invalidates keepAlive provider | WIRED | `ref.invalidate(mapStyleSourcesProvider)` + `mapStyleJsonProvider` in `_retryOnline` |
| `list_screen.dart` | `listSearchProvider` | Try again invalidates search provider | WIRED | `ref.invalidate(listSearchProvider)` in `_retryOnline`, plus pre-existing pull-to-refresh call (2 occurrences total) |
| `wanderer_layout.dart` | `avatar_cache_util.dart` | FileImage over cached file | WIRED | `_NavAvatar` uses `cachedAvatarFile` + `FileImage` |
| `profile_screen.dart` | `authProvider` UserEntity | Offline fallback scaffold | WIRED | `_buildCachedProfile(cachedUser)` built from `ref.watch(authProvider).value` |
| `settings_account_screen.dart` | `offline_guard_util.dart` | guardOnline early-return in mutating handlers | WIRED | 3 call sites confirmed by grep |
| `settings_privacy_screen.dart` (+ notifications/language/categories/subcategories) | `offline_guard_util.dart` | guardOnline in save chokepoint (+ reorder for categories/subcategories) | WIRED | grep-confirmed on all 5 remaining sub-screens |
| `settings_offline_banner.dart` | `onlineStatusProvider` | ref.watch driving visibility | WIRED | `ref.watch(onlineStatusProvider)` in `SettingsOfflineBanner.build` |

### Behavioral Spot-Checks / Automated Verification

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| flutter analyze on all 24 files touched by this task | `flutter analyze <24 files>` | "No issues found! (ran in 5.0s)" | PASS |
| online_status_provider_test.dart (isConnectionFailure + notifier cases) | `flutter test test/provider/online_status_provider_test.dart` | 11/11 passed | PASS |
| wanderer_offline_state_test.dart (render/retry/in-flight/no-exception-text) | `flutter test test/components/wanderer_offline_state_test.dart` | all cases passed | PASS |
| avatar_cache_util_test.dart (sanitization/stability/null-handling) | `flutter test test/util/avatar_cache_util_test.dart` | 7/7 passed | PASS |
| offline_guard_util_test.dart (guard behavior + toast queueing) | `flutter test test/util/offline_guard_util_test.dart` | all cases passed | PASS |
| Full-suite regression (`flutter test`) | `flutter test` | 451 passed, 4 failed — all 4 in `test/components/route_planner/settings_tab_test.dart`, confirmed pre-existing/unrelated (caused by separately-committed `28cabf9d "Fix route planner leg calculation"`, not touching any file in this task's scope) | PASS (matches SUMMARY's documented baseline) |
| No stray `isBackendReachable` references outside its definition/caller | `grep -rn 'isBackendReachable' lib/` filtered | 0 matches outside `connectivity_util.dart` / `online_status_provider.dart` | PASS |
| No new package added | `grep -c cached_network_image pubspec.yaml` | 0 | PASS |
| No ObjectBox schema change | `git log` on `objectbox-model.json`/`user_entity.dart` across task commits | no touches by d1ba7a1e..8ecbbd9f | PASS |
| Appearance screen untouched (local-only, no guard) | `grep guardOnline/setThemeMode settings_appearance_screen.dart` | 0 guardOnline, 1 setThemeMode | PASS |
| Offline-regions screen not double-bannered | `grep SettingsOfflineBanner settings_offline_regions_screen.dart` | 0 | PASS |
| Anti-pattern scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) on all 22 modified files | grep -i | 0 matches | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| OFFLINE-STATUS | 01 | Single source of truth for online/offline | SATISFIED | `onlineStatusProvider` + `InterceptorsWrapper` |
| OFFLINE-MAP | 01 | Map screen offline takeover | SATISFIED | `map_screen.dart` truth #1, #3 |
| OFFLINE-LIST | 01 | List screen offline takeover | SATISFIED | `list_screen.dart` truth #2, #3 |
| OFFLINE-AVATAR | 01 | Cached nav avatar | SATISFIED | `wanderer_layout.dart` truth #4 |
| OFFLINE-PROFILE | 01 | Cached profile + reachable settings | SATISFIED | `profile_screen.dart` truth #5 |
| OFFLINE-SETTINGS | 01 | Settings browsable + gated mutations | SATISFIED | Tasks 4/5, truths #7, #8 |

No orphaned requirements found in REQUIREMENTS.md for this quick task (quick tasks do not use REQUIREMENTS.md phase mapping).

### Anti-Patterns Found

None. No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers, no stub returns, no discarded-return error handlers remaining in any of the 22 files modified by this task.

### Human Verification Required

None required to pass automated verification — the plan's own `<verification>` section documents a "Device sanity pass" (airplane-mode walkthrough) as **non-blocking** and the SUMMARY explicitly records it was not performed in the executor session, recommending it as a manual follow-up. This is consistent with the plan's own design (it explicitly marks that pass "not a blocking gate"), so it does not change the automated status. Recommended before considering the on-device UX fully validated:

### 1. Airplane-mode device walkthrough

**Test:** Launch online, sign in, visit profile once (caches avatar), enable airplane mode, force-quit, relaunch. Visit map, list, nav avatar, profile, and each of the 7 settings tiles. Disable airplane mode and tap Try again on the map.
**Expected:** Map/list show the offline takeover with working sheet/retry; nav avatar and profile show cached identity; every settings tile opens and renders rows with a banner (except Appearance, which is unaffected); reconnecting and retrying repopulates the map without a restart.
**Why human:** Requires a physical/simulator airplane-mode toggle, app restart, and visual confirmation of layout/animation — not verifiable via static analysis or headless tests.

### Gaps Summary

No gaps found. All 8 must-have truths are verified against actual code (not just SUMMARY claims), all 7 required artifacts exist and are substantive (not stubs), all 8 key links are wired, `flutter analyze` is clean across all 24 touched files, all 4 new/modified test files pass in full, and the full-suite regression run shows exactly the pre-existing, out-of-scope failures the SUMMARY documented (4 in `settings_tab_test.dart`, unrelated route-planner work) with no new regressions introduced by this task's 5 commits. No new package was added and no ObjectBox schema change occurred, matching the plan's constraints.

---

*Verified: 2026-07-29T15:40:00Z*
*Verifier: Claude (gsd-verifier)*
