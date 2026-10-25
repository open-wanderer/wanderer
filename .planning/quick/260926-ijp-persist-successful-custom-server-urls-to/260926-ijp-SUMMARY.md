---
phase: quick-260926-ijp
plan: 01
subsystem: app (Flutter instance picker / auth)
tags: [flutter, riverpod, objectbox, l10n, auth]
dependency-graph:
  requires: []
  provides:
    - "pushRecentServer/encodeRecentServers/decodeRecentServers (app/lib/util/recent_servers.dart)"
    - "recentServersProvider (app/lib/provider/welcome/recent_servers_provider.dart)"
    - "LocalSettingsEntity.recentServersJson"
  affects:
    - "app/lib/provider/auth_provider.dart (login/loginWithOAuth success hook)"
    - "app/lib/routes/server_selection_screen.dart (picker UI)"
tech-stack:
  added: []
  patterns:
    - "read-modify-write on the LocalSettingsEntity singleton row (matches local_settings_provider.dart / map_source_persistence.dart)"
    - "tolerant JSON codec that never throws on malformed persisted data"
key-files:
  created:
    - app/lib/util/recent_servers.dart
    - app/test/util/recent_servers_test.dart
    - app/lib/provider/welcome/recent_servers_provider.dart
    - app/lib/provider/welcome/recent_servers_provider.g.dart
    - app/test/provider/server_selection_provider_test.dart
  modified:
    - app/lib/entities/local_settings_entity.dart
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json
    - app/lib/provider/welcome/server_selection_provider.dart
    - app/lib/provider/welcome/server_selection_provider.g.dart
    - app/lib/provider/auth_provider.dart
    - app/lib/provider/auth_provider.g.dart
    - app/lib/routes/server_selection_screen.dart
    - app/test/routes/server_selection_screen_test.dart
    - app/lib/i18n/app_*.arb (14 files)
    - app/lib/i18n/app_localizations*.dart (regenerated)
decisions:
  - "Used git add -p to stage only this plan's hunks in auth_provider.dart, leaving a concurrent uncommitted debug-session change (cancelPendingRequests / ref.mounted guard in logout()/_updateUserEntity, referenced by .planning/debug/app-relogin-after-logout.md) untouched and still uncommitted in the working tree, per the scope boundary and the instruction to never stage unrelated uncommitted work."
  - "Disabled Riverpod's default AsyncNotifier retry (10 exponential-backoff attempts, up to 6.4s each) in the new provider test via ProviderContainer(retry: (_, __) => null) -- without it, a throwing build() never settles to AsyncError within the test's own timeout."
metrics:
  duration: ~45min
  completed: 2026-09-26
---

# Quick Task 260926-ijp: Persist successful custom server URLs to a "Last used" picker section Summary

Added a persisted "Last used" instance list (max 2, most recent first, deduped by normalized URL) to the Flutter app's server picker, recorded only on successful password/OAuth/register sign-in — never on mere selection, a failed login, `refresh()`, or the cold-start background validation.

## What Was Built

**Task 1 — Pure recording logic** (`app/lib/util/recent_servers.dart`, TDD): `kMaxRecentServers = 2`, `pushRecentServer` (normalize, dedupe-by-lowercased-URL, move-to-top, cap, keep-existing-metadata-on-nameless-match), `encodeRecentServers`/`decodeRecentServers` (tolerant codec — never throws, drops malformed/duplicate/unnormalizable elements). 20 unit tests, no Flutter/ObjectBox imports.

**Task 2 — Persistence, provider, auth hook** (TDD):
- `LocalSettingsEntity.recentServersJson` (device-level field, empty default, survives logout/`account_data_purge`).
- `RecentServersNotifier` / `recentServersProvider` (`keepAlive`, read-modify-write on the singleton row).
- `ServerSelectionNotifier.setSelectedServer` now falls back to `state.value?.availableServers ?? const []` instead of `state.requireValue...`, so a "Last used" tap works while servers.json is loading or has failed. Covered by a new provider test; the throwing-stub case required disabling Riverpod's default retry policy in the test container to avoid a 30s+ timeout.
- `Auth._recordLastUsedServer()`: best-effort, try/catch-wrapped, called only from inside `login()`'s and `loginWithOAuth()`'s `AsyncValue.guard` bodies immediately after `_updateUserEntity` succeeds. Not called from `_updateUserEntity`, `refresh()`, `_validateInBackground()`, or `register()` directly (it signs in through `login()`).
- `dart run build_runner build --delete-conflicting-outputs` regenerated `objectbox.g.dart`/`objectbox-model.json` (additive property, new UID) and `recent_servers_provider.g.dart`.

**Task 3 — Picker UI + l10n**:
- Inserted `"last_used"` into all 14 ARB files (en/de/cs get real translations; the other 11 carry the English copy, matching the existing `select_instance` precedent) and ran `flutter gen-l10n` (`untranslated_messages.json` stayed `{}`).
- `server_selection_screen.dart`: extracted a shared `_buildServerTile` (nameless-safe title fallback via `_displayUrl`, image-less-safe leading via a shared fallback icon builder), hardened the remote-list filter (`s.name!` → `(s.name ?? '')`), and replaced the fixed `Column`/`ListView` body with an `Expanded CustomScrollView` of slivers: a "Last used" header + tiles + divider (shown only when non-empty, not filtered by search, independent of the servers.json `AsyncValue`), followed by the existing data/no-match/loading/error branches as `SliverList`/`SliverFillRemaining`.
- Widget tests: hidden-when-empty, header+tiles+nameless-title for recorded servers, renders alongside a never-completing loading spinner (`pump()`, not `pumpAndSettle()`), and a nameless remote-list entry renders with no null-bang crash.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — blocking] Riverpod's default AsyncNotifier retry policy stalled the new provider test**
- **Found during:** Task 2, writing `server_selection_provider_test.dart`
- **Issue:** A throwing `build()` never settles to `AsyncError` within a reasonable time — Riverpod 3's default `retry` (`ProviderContainer.defaultRetry`) retries up to 10 times with exponential backoff (200ms → 6.4s cap), which exceeds the test framework's 30s default timeout well before giving up.
- **Fix:** Passed `retry: (retryCount, error) => null` to the `ProviderContainer` in the throwing-stub test only, documented inline with a comment explaining why.
- **Files modified:** `app/test/provider/server_selection_provider_test.dart`
- **Commit:** `153b58e7`

### Working-tree isolation (not a plan deviation, but notable)

A concurrent, uncommitted debug-session change was present in the working tree in two files this plan also had to touch (`app/lib/provider/auth_provider.dart`) or read (`app/lib/provider/api_provider.dart`): a `cancelPendingRequests()`/`CancelToken` mechanism and an `!ref.mounted` guard in `_updateUserEntity`, related to `.planning/debug/app-relogin-after-logout.md`. This is unrelated to LASTUSED-01..04 and was never part of this plan's `files_modified`.

`git add -p` was used to stage only this plan's 4 hunks in `auth_provider.dart` (imports, `_recordLastUsedServer`, and its two call sites), leaving the debug session's 2 hunks (in `logout()` and `_updateUserEntity`) unstaged and uncommitted, exactly as they were found. `api_provider.dart`/`api_provider.g.dart` (entirely the debug session's own file) were left completely untouched. `build_runner`'s regeneration also touched several unrelated `.g.dart` files (`map_style_json_provider.g.dart`, `map_style_sources_provider.g.dart`, `region/tile_proxy_provider.g.dart`, `router_provider.g.dart`) purely as hash/doc-comment echoes of other pre-existing uncommitted source edits; none of these were staged or committed either.

One side effect: the committed `auth_provider.g.dart`'s embedded source hash was computed by `build_runner` against the full disk state (this plan's hunks + the still-uncommitted debug hunks), so it won't exactly match the git-committed `auth_provider.dart` until the debug session's own changes are eventually committed too. This is inert — the hash is only used by `build_runner` to detect staleness for future regeneration, not at runtime — and will self-correct on the next `build_runner build`.

## Known Stubs

None.

## Threat Flags

None. All work matches the plan's threat register (T-ijp-01..05); no new trust boundaries or surfaces were introduced.

## Verification

- `flutter test` (full suite): 1176 passed, 1 skipped, 0 failed.
- `flutter analyze`: 13 pre-existing info-level issues, all in files untouched by this plan (`lib/entities/actor_entity.dart`, `lib/entities/category_entity.dart`, `lib/provider/navigation_stats_provider.dart`, `lib/store/local_photo_store.dart`, `lib/util/local/id.dart`, `vendor/tiptap_flutter/...`) — none introduced by this plan.
- All 14 ARB files carry exactly one `last_used` key; `untranslated_messages.json` is `{}`.
- `grep -c "_recordLastUsedServer()" app/lib/provider/auth_provider.dart` → 3 (definition + 2 call sites); absent from `_updateUserEntity`, `refresh()`, `_validateInBackground()`, `register()`.
- `objectbox-model.json`/`objectbox.g.dart` contain `recentServersJson`, diff scoped to the new additive property.

Manual on-device verification (deferred to the user, per plan):
1. Pick a custom URL and log in; log out; reopen the picker — "Last used" shows that server.
2. Log into a remote-list server, then log out — both entries show, newest first.
3. Log into a third server — only the two newest remain.
4. Kill and relaunch the app — the list persists.

## Self-Check: PASSED

All 8 created/modified source and test files confirmed present on disk. All 6 commit hashes (`81923e64`, `da2ae1da`, `153b58e7`, `62f736de`, `adfd4a06`, `e59f35e5`) confirmed in `git log`.
