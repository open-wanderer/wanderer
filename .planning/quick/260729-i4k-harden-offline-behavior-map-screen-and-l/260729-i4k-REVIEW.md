---
phase: 260729-i4k-harden-offline-behavior-map-screen-and-l
reviewed: 2026-07-29T13:06:22Z
depth: quick
files_reviewed: 26
files_reviewed_list:
  - app/lib/components/base/wanderer_layout.dart
  - app/lib/components/base/wanderer_offline_state.dart
  - app/lib/components/settings/email_change_sheet.dart
  - app/lib/components/settings/password_change_sheet.dart
  - app/lib/components/settings/settings_offline_banner.dart
  - app/lib/main.dart
  - app/lib/provider/api_provider.dart
  - app/lib/provider/auth_provider.dart
  - app/lib/provider/online_status_provider.dart
  - app/lib/routes/list_screen.dart
  - app/lib/routes/map_screen.dart
  - app/lib/routes/profile_screen.dart
  - app/lib/routes/settings_account_screen.dart
  - app/lib/routes/settings_categories_screen.dart
  - app/lib/routes/settings_language_screen.dart
  - app/lib/routes/settings_notifications_screen.dart
  - app/lib/routes/settings_privacy_screen.dart
  - app/lib/routes/settings_subcategories_screen.dart
  - app/lib/routes/trail_source_select_screen.dart
  - app/lib/util/avatar_cache_util.dart
  - app/lib/util/connectivity_util.dart
  - app/lib/util/offline_guard_util.dart
  - app/test/components/wanderer_offline_state_test.dart
  - app/test/provider/online_status_provider_test.dart
  - app/test/util/avatar_cache_util_test.dart
  - app/test/util/offline_guard_util_test.dart
findings:
  critical: 3
  warning: 3
  info: 2
  total: 8
status: issues_found
---

# Phase 260729-i4k: Code Review Report

**Reviewed:** 2026-07-29T13:06:22Z
**Depth:** quick (escalated to file-level reading where grep alone was insufficient to validate offline-state gating logic)
**Files Reviewed:** 26
**Status:** issues_found

## Summary

This change adds an `onlineStatusProvider` fed by a Dio interceptor, a `guardOnline()` pre-flight gate for mutating actions, a shared `WandererOfflineState` full-takeover widget, and cached-avatar/cached-identity fallbacks for the bottom nav and profile screen. The plumbing (`isConnectionFailure`, the interceptor, `guardOnline`, the avatar-cache path-traversal guard) is well-tested and sound.

However, three of the newly-added offline-state code paths have real correctness bugs: the profile screen's cached-identity fallback mislabels an ordinary in-progress fetch as "offline" (shown even while genuinely online), the map screen's retry handler operates on a stale/disposed map controller reference, and the account-settings avatar upload can call `setState()` after the widget is disposed. There is also a gap in the map screen's offline takeover: the trail-results sheet stays visible and interactive on top of the "you're offline" message instead of being suppressed with the rest of the map UI.

## Critical Issues

### CR-01: Profile screen shows "You're offline" while genuinely online and merely loading

**File:** `app/lib/routes/profile_screen.dart:87-98` (fallback renderer at `92-160`)
**Issue:** The own-profile branch renders `_buildCachedProfile(cachedUser)` for *both* the `loading` and `error` states of `actorAsync`:
```dart
if (isOwn) {
  final cachedUser = ref.watch(authProvider).value;
  return Scaffold(
    body: actorAsync.when(
      data: (actor) => _buildProfile(actor),
      loading: () => cachedUser != null
          ? _buildCachedProfile(cachedUser)
          : const Center(child: CircularProgressIndicator()),
      error: (err, stack) => cachedUser != null
          ? _buildCachedProfile(cachedUser)
          : WandererError(err: err, stack: stack),
    ),
  );
}
```
`_buildCachedProfile` (line 122) unconditionally renders a `WandererOfflineState` with `l10n.offline_title` / `l10n.offline_profile_body` ("You're offline...") in its `SliverFillRemaining`, with no check against `onlineStatusProvider`:
```dart
Widget _buildCachedProfile(UserEntity user) {
  final isOnline = ref.watch(onlineStatusProvider); // only used for the avatar
  return CustomScrollView(
    slivers: [
      ...
      SliverFillRemaining(
        hasScrollBody: false,
        child: WandererOfflineState(
          title: l10n.offline_title,
          body: l10n.offline_profile_body, // "you are offline" copy
          ...
        ),
      ),
    ],
  );
}
```
Because a cached `UserEntity` exists for almost any user who has logged in before, this fires on the very first paint of the Profile tab after every app launch (while `ownProfileProvider`'s network fetch is merely in flight) and after every `ref.invalidate(ownProfileProvider)` elsewhere in the app (e.g. `settings_account_screen.dart:55` after an avatar upload, or `settings_account_screen.dart:112` after a bio save) — all cases where the device is fully online. The user sees an incorrect "You're offline" message with a "Try again" button during completely normal loading, not just when actually disconnected.
**Fix:** Gate the offline-copy branch on `isOnline`; render a lightweight loading indicator (or the existing spinner) instead when online-but-loading:
```dart
SliverFillRemaining(
  hasScrollBody: false,
  child: isOnline
      ? const Center(child: CircularProgressIndicator())
      : WandererOfflineState(
          title: l10n.offline_title,
          body: l10n.offline_profile_body,
          retryLabel: l10n.offline_try_again,
          onRetry: () async {
            final online =
                await ref.read(onlineStatusProvider.notifier).refresh();
            if (online) ref.invalidate(ownProfileProvider);
          },
        ),
),
```

### CR-02: `_retryOnline()` on the map screen operates on a stale/disposed `MapController`

**File:** `app/lib/routes/map_screen.dart:218-231`, `363-378`
**Issue:** When `isOnline` becomes `false`, the whole `TrailCollectionMap` (and the native map view it owns) is swapped out of the tree for `WandererOfflineState`:
```dart
if (!isOnline)
  Container(..., child: WandererOfflineState(..., onRetry: _retryOnline))
else
  TrailCollectionMap(
    ...
    onMapCreated: (controller) => _controller = controller,
    ...
  ),
```
`_controller` (declared `ml.MapController? _controller;` at line 57) is never reset to `null` anywhere — there is no `onMapDisposed`/teardown hook clearing it when `TrailCollectionMap` is removed from the tree. `_retryOnline()` reads `_controller` and immediately calls native methods on it:
```dart
Future<void> _retryOnline() async {
  final online = await ref.read(onlineStatusProvider.notifier).refresh();
  if (!online) return;
  ref.invalidate(mapStyleSourcesProvider);
  ref.invalidate(mapStyleJsonProvider);
  final controller = _controller;
  if (controller == null) return;
  final bounds = controller.getVisibleRegion();   // stale controller
  final zoom = controller.getCamera().zoom;        // stale controller
  ref.read(mapClusterSearchProvider.notifier).searchInBounds(bounds, zoom);
  ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
}
```
At the point `_retryOnline` runs (triggered by the retry button while the offline takeover is still showing), the widget has not yet rebuilt with the online branch, so `_controller` is still the reference from *before* the map went offline. Calling `getVisibleRegion()`/`getCamera()` on a torn-down native map controller risks a platform-channel exception, or at best silently uses stale bounds/zoom — defeating the entire purpose of this retry path (re-running the last search after reconnecting).
**Fix:** Null out `_controller` in a teardown hook when the map is torn down (e.g. wrap in a `didUpdateWidget`/`AnimatedSwitcher` dispose callback, or track a `_mapReady` completer set in `onMapCreated` and awaited before use), and have `_retryOnline` wait for a fresh `onMapCreated` callback before reading bounds/zoom instead of trusting a controller captured before the offline swap.

### CR-03: Avatar upload can call `setState()` after the widget is disposed

**File:** `app/lib/routes/settings_account_screen.dart:43-72`
**Issue:** Every other exit path in `_pickAndUploadAvatar` checks `context.mounted` before touching state, but the `finally` block does not:
```dart
try {
  ...
  await ref.read(apiProvider).post('/user/$userId/file', data: formData);
  if (!context.mounted) return;
  await ref.read(authProvider.notifier).refresh();
  ref.invalidate(ownProfileProvider);
} catch (_) {
  if (!context.mounted) return;
  ...
} finally {
  setState(() {                 // <-- no mounted guard
    _avatarLoading = false;
  });
}
```
If the user navigates away from `SettingsAccountScreen` while the upload/refresh is in flight, `finally` still runs and calls `setState()` on a disposed `State`, throwing `FlutterError: setState() called after dispose()`.
**Fix:**
```dart
} finally {
  if (mounted) {
    setState(() {
      _avatarLoading = false;
    });
  }
}
```

## Warnings

### WR-01: Map screen's trail-results sheet is not hidden while offline, contradicting the offline takeover

**File:** `app/lib/routes/map_screen.dart:361-372` vs `610-643`
**Issue:** The map itself is replaced by a full-area `WandererOfflineState` when `!isOnline` (line 363), and the FAB / "search this area" button are explicitly hidden with `if (_selectedTrail == null && isOnline)` (lines 520, 568) — but the `DraggableScrollableSheet` showing the trail list (`key: const ValueKey('trail_sheet')`, line 611) has no `isOnline` guard at all and is always rendered as a Stack sibling on top of the offline message. `WandererOfflineState`'s own docstring describes it as "a full-area takeover ... for screens that have nothing to show", but here it is drawn *underneath* a still-interactive sheet showing the last cached trail search results. Tapping a stale `TrailCard` still calls `_selectTrail` → `ref.read(trailPolylineProvider(trailId).future)`, a network fetch that will hang/fail while offline — exactly the class of interaction this feature set is meant to prevent.
**Fix:** Gate the sheet's rendering (or at least disable `onTrailSelect`) on `isOnline`, consistent with the FAB and search-this-area button:
```dart
if (isOnline)
  Opacity(
    key: const ValueKey('trail_sheet'),
    ...
  ),
```

### WR-02: `password_change_sheet.dart` reads `BuildContext` before the post-await `mounted` check

**File:** `app/lib/components/settings/password_change_sheet.dart:64-74`
**Issue:** In the `on DioException catch (error)` block, the fallback error message reads `AppLocalizations.of(context)!` *before* the `if (!mounted) return;` guard that every other branch in this file (and in the sibling `email_change_sheet.dart`) observes:
```dart
} on DioException catch (error) {
  String message;
  try {
    final apiError = ApiError.fromJson(error.response?.data);
    message = apiError.message;
  } catch (_) {
    message =
        error.message ??
        AppLocalizations.of(context)!.error_updating_password; // used pre-mounted-check
  }
  if (!mounted) return;
  ...
}
```
If the sheet is dismissed while the request is in flight (an async gap already occurred at the awaited `post(...)` call) and the response body also fails to parse as `ApiError`, this looks up an inherited widget on a deactivated `BuildContext`, which can throw ("Looking up a deactivated widget's ancestor is unsafe") in debug builds.
**Fix:** Move the `if (!mounted) return;` check to the top of the `catch` block, before building `message`, or capture `l10n` once at the top of `_submit()` before any `await`.

### WR-03: `main.dart` leaves commented-out debug code in place

**File:** `app/lib/main.dart:41`
**Issue:**
```dart
final store = await openStore(directory: dbPath);
final proxyServer = await TileProxyServer.start(store);

// store.box<TrailEntity>().removeAll();
```
A leftover debug line for wiping the local trail cache is committed directly in `main()`. It is dead code that risks being uncommented accidentally in a future edit (data loss for every user on next app start).
**Fix:** Delete the line; if a "clear local cache" affordance is needed, wire it into a settings/debug screen instead of leaving it commented in `main()`.

## Info

### IN-01: Re-reads `authProvider` a second time instead of reusing the already-validated value

**File:** `app/lib/routes/settings_categories_screen.dart:531,550` and `app/lib/routes/settings_subcategories_screen.dart:510,529`
**Issue:** Both screens validate `ref.read(authProvider).value?.preferredUsername` for null and, a few lines later, force-unwrap a *second*, independent read of the same provider:
```dart
final username = ref.read(authProvider).value?.preferredUsername;
if (username == null) { ...; return; }
...
final handle = '@${ref.read(authProvider).value!.preferredUsername}'; // re-read + force unwrap
```
No `await` occurs between the two reads today, so this happens to be safe, but it duplicates the lookup and force-unwraps rather than reusing the already-null-checked `username` local — a maintenance hazard if an `await` is ever introduced between the two lines.
**Fix:** Reuse the validated `username` local: `final handle = '@$username';`

### IN-02: `debugPrint` swallow-and-log pattern is consistent but untested

**File:** `app/lib/util/avatar_cache_util.dart:82-84`, `app/lib/routes/map_screen.dart:387-388`
**Issue:** Both call sites intentionally swallow all exceptions and log via `debugPrint`, which is consistent with the project's "best-effort, never throw" design goal for these paths, but neither failure path is covered by a test that forces the download/cluster-layer call to throw and asserts the caller still completes normally.
**Fix:** Not blocking; consider adding a regression test that injects a failing `Dio`/style call to lock in the "never throws" contract.

---

_Reviewed: 2026-07-29T13:06:22Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
