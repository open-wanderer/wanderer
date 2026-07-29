# Quick Task 260729-i4k: Harden offline behavior - Research

**Researched:** 2026-07-29
**Domain:** Flutter / Riverpod 3 codegen + dio interceptors + offline UI states
**Confidence:** HIGH (all findings verified by reading installed package source and the actual app source)

## Summary

Everything needed is already in the repo — no new packages are required for the online-status
provider, the offline takeover UIs, or the profile-screen fallback. The one genuine gap is the
bottom-nav avatar: `NetworkImage` gives **memory-only** caching (lost on app restart), and the
app has **no disk image cache** and no `cached_network_image` dependency, so "avatar shown
offline" needs either a small avatar-to-disk download (mirroring the existing
`trail_download_service.dart` photo-download pattern, which already writes files and reads them
back via `FileImage`) or a new package.

The circular-dependency worry in the brief is **not** a Riverpod graph cycle: `Ref.read` does
*not* register a dependency edge (only `watch`/`listen` do — verified in
`riverpod-3.2.1/lib/src/core/ref.dart:537`). The real hazard is *mutual synchronous
initialization* (Api.build → OnlineStatus.build → Api.build). Avoid it by (a) making
`OnlineStatus.build()` return a plain `true` with zero provider reads, and (b) doing
`ref.read(onlineStatusProvider.notifier)` **inside the interceptor callbacks**, not at
`Api.build()` time.

**Primary recommendation:** Add `lib/provider/online_status_provider.dart` as a
`@Riverpod(keepAlive: true) class OnlineStatus extends _$OnlineStatus` with `bool build() => true`,
register an `InterceptorsWrapper` in `Api.build()` that marks online on `onResponse` **and** on
`badResponse` errors, marks offline only on `connectionError | connectionTimeout | (unknown &&
error is SocketException)`; change `isBackendReachable` to take a `Dio` instead of a `WidgetRef`
so both the notifier (`Ref`) and any widget can call it.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- Global Riverpod `onlineStatusProvider` (keepAlive), single source of truth, seeded at launch by
  `isBackendReachable`, kept live by a dio interceptor on the shared api client.
- Successful response ⇒ online. Connection-level failure ⇒ offline. **Ordinary HTTP error
  responses (404/422/500) must NOT flip to offline.**
- Migrate `lib/main.dart:240`, `lib/main.dart:287`, `lib/routes/trail_source_select_screen.dart:99`
  to the provider. Keep `isBackendReachable` as the underlying probe — do not delete it.
- Map screen: full-screen takeover of the **map area only**; the `DraggableScrollableSheet` stays
  intact and usable.
- List screen: full takeover replacing the generic "Something went wrong".
- Profile: bottom-nav avatar + basic profile info reconstructed from the local `UserEntity`.
  Network-only extras ⇒ "You are offline" empty state, not an error.
- Settings visible and browsable offline; network-requiring actions disabled/deferred. Settings
  must always be reachable from the profile screen even offline.

### Claude's Discretion
- Exact copy/styling of the offline message and Try again button (follow existing app conventions).
- Per-row disabling vs. a single settings banner — whichever matches existing patterns with the
  least structural change.

### Deferred Ideas (OUT OF SCOPE)
- None listed.

## Project Constraints (from CLAUDE.md)

- Flutter app lives under `app/` — Dart only for this task; no SvelteKit/Go changes.
- Riverpod providers are the state layer; components guard against missing data.
- No new linter config; 2-space indent for TS, `dart format` defaults for Dart.
- GSD workflow: this is a `/gsd-quick` task.
- Memory note (user-level): use `.value`, not `.valueOrNull`, for nullable `AsyncValue` access.

## Finding 1 — Precise dio condition (VERIFIED against installed source)

`DioExceptionType` in `~/.pub-cache/hosted/pub.dev/dio-5.9.2/lib/src/dio_exception.dart:15-41`
[VERIFIED: local package source] has exactly these values:
`connectionTimeout, sendTimeout, receiveTimeout, badCertificate, badResponse, cancel,
connectionError, unknown`.

`io_adapter.dart:113` catches `SocketException` and `io_adapter.dart:131` throws
`DioException.connectionError(...)` [VERIFIED: local package source] — so on mobile, a real
"no route to host" / DNS failure surfaces as `DioExceptionType.connectionError`, **not**
`unknown`. The `unknown && error is SocketException` arm is a belt-and-braces catch for
non-adapter paths.

**Use exactly this condition:**

```dart
// lib/provider/online_status_provider.dart (or inline in api_provider.dart)
import 'dart:io' show SocketException;

/// True only when the request never reached a server. A `badResponse`
/// (404/422/500/503) proves the backend IS reachable and must read as online.
bool isConnectionFailure(DioException e) =>
    e.type == DioExceptionType.connectionError ||
    e.type == DioExceptionType.connectionTimeout ||
    (e.type == DioExceptionType.unknown && e.error is SocketException);
```

Deliberately **excluded** and why:
| Type | Excluded because |
|------|------------------|
| `badResponse` | Server answered — the CONTEXT decision explicitly requires online. |
| `badCertificate` | TLS handshake means a server was reached. |
| `sendTimeout` / `receiveTimeout` | Cannot fire: `Api.build()` sets **only** `connectTimeout` (`lib/provider/api_provider.dart:20-25`, with an explicit comment saying a `receiveTimeout` would abort large tile downloads). |
| `cancel` | User/app-initiated; says nothing about connectivity. |

`Interceptor` callback signatures (`dio-5.9.2/lib/src/interceptor.dart:209-230`)
[VERIFIED: local package source]:
```dart
void onRequest(RequestOptions options, RequestInterceptorHandler handler);
void onResponse(Response response, ResponseInterceptorHandler handler);
void onError(DioException err, ErrorInterceptorHandler handler);
```
Always call `handler.next(...)` — dropping it swallows the response/error.

`dart:io` is safe here: the app has **no `web/` directory** (verified — `app/` contains only
`android`, `ios`, `lib`, `assets`, `test`, `vendor`), so it is Android/iOS only.

## Finding 2 — Riverpod wiring, and why there is no cycle

**`Ref.read` creates no dependency edge.** `riverpod-3.2.1/lib/src/core/ref.dart:537-546`
[VERIFIED: local package source]:
```dart
StateT read<StateT>(ProviderListenable<StateT> listenable) {
  _throwIfInvalidUsage();
  final result = container.read(listenable);
  if (kDebugMode) _debugAssertCanDependOn(listenable);   // scoping check only
  return result;
}
```
Compare `watch` (line 652) which calls `_element.listen(...)` and *does* create an edge. So
`Api` ⇄ `OnlineStatus` via `ref.read` cannot trigger Riverpod's cycle assertion.

**The real hazard is mutual synchronous init.** If `OnlineStatus.build()` read `apiProvider`
*and* `Api.build()` read `onlineStatusProvider`, whichever initializes first re-enters the other
mid-build. Two rules eliminate it:
1. `OnlineStatus.build()` returns a literal (`true`) and touches **no** other provider.
2. `Api.build()` does **not** resolve the notifier at build time — the interceptor closure calls
   `ref.read(onlineStatusProvider.notifier)` lazily, per request.

**Recommended shape** (matches existing codegen style in `lib/provider/map_camera_provider.dart`
and `lib/provider/toast_provider.dart`):

```dart
// lib/provider/online_status_provider.dart
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/connectivity_util.dart';

part 'online_status_provider.g.dart';

@Riverpod(keepAlive: true)
class OnlineStatus extends _$OnlineStatus {
  /// Optimistic default: no provider reads here — see Pitfall 1.
  @override
  bool build() => true;

  void markOnline()  { if (!state) state = true;  }
  void markOffline() { if (state)  state = false; }

  /// Explicit probe. Used to seed at launch and to back the "Try again" CTA.
  Future<bool> refresh() async {
    final ok = await isBackendReachable(ref.read(apiProvider));
    state = ok;
    return ok;
  }
}
```

```dart
// lib/provider/api_provider.dart — inside Api.build(), after the CookieManager
dio.interceptors.add(
  InterceptorsWrapper(
    onResponse: (response, handler) {
      ref.read(onlineStatusProvider.notifier).markOnline();
      handler.next(response);
    },
    onError: (err, handler) {
      final notifier = ref.read(onlineStatusProvider.notifier);
      isConnectionFailure(err) ? notifier.markOffline() : notifier.markOnline();
      handler.next(err);
    },
  ),
);
```

**`isBackendReachable` signature change is required.** It currently takes `WidgetRef`
(`lib/util/connectivity_util.dart:16`). `Ref` and `WidgetRef` are unrelated types in Riverpod 3 —
there is no shared supertype. Since all three existing call sites are being migrated away, the
only remaining caller is the notifier. Change it to take the `Dio` directly:

```dart
Future<bool> isBackendReachable(Dio api) async {
  try {
    final res = await api.get('/health').timeout(const Duration(seconds: 5));
    return res.statusCode == 200;
  } catch (_) {
    return false;
  }
}
```
This also drops the `flutter_riverpod` import from the util, making it trivially unit-testable.

**Seeding at launch:** call from `_MainAppState.initState` in `lib/main.dart`
(`unawaited(ref.read(onlineStatusProvider.notifier).refresh())`) alongside the existing
`_authSub`/`_shareSub` setup at `lib/main.dart:78-115`. The first `state =` write happens after an
`await`, i.e. after the build phase — see Pitfall 2.

## Finding 3 — Map screen: exact structure and where the takeover goes

**File:** `app/lib/routes/map_screen.dart` (931 lines). Build method starts at line 211 and returns
a `Stack` at **line 337** with these children in order:

| Lines | Child | Offline treatment |
|-------|-------|-------------------|
| 339-478 | `TrailCollectionMap(...)` — the map itself | **Replace this widget** with the offline takeover |
| 483-529 | "center on my location" FAB (rides with sheet) | Hide when offline |
| 531-571 | "Search this area" button | Hide when offline |
| **573-707** | `Opacity(key: ValueKey('trail_sheet'), child: DraggableScrollableSheet(...))` | **Must survive** — CONTEXT decision |
| 709-819 | Search bar + filter/sort chips (SafeArea/topCenter) | Optional: hide or leave |
| 821-844 | "Map" button that collapses the sheet | Keep |
| 846-867 | Selected-trail `TrailListItem` | Only shows when a trail is selected |

The takeover replaces **only the `TrailCollectionMap` at lines 339-478**, leaving the sheet
(573-707) untouched. Because the sheet is a later `Stack` child it already paints on top, so a
plain swap works with no reordering.

**Why the map currently breaks offline (root cause, VERIFIED by reading source):**
`TrailCollectionMap` (`lib/components/base/trail_collection_map.dart:73-83`) watches
`mapStyleJsonProvider`, which awaits `mapStyleSourcesProvider`
(`lib/provider/map_style_sources_provider.dart:10-13` → `api.get('/map/style-sources')`). Offline
that throws, and `trail_collection_map.dart:80` renders a bare
`Center(child: Text(error.toString()))` — a raw exception string on screen. Both providers are
`@Riverpod(keepAlive: true)`.

**Consequence for the "Try again" CTA:** `mapStyleSourcesProvider` is keepAlive and caches its
error, so recovery requires `ref.invalidate(mapStyleSourcesProvider)` in addition to the
`onlineStatusProvider.notifier.refresh()`. Also re-trigger the searches (`mapClusterSearchProvider`
/ `mapTrailSearchProvider` `.searchInBounds(...)`, called at map_screen.dart:364-369) or the sheet
stays empty after reconnect.

**List screen:** `app/lib/routes/list_screen.dart` — 140 lines, structure is simple.
`AsyncLoader<ListSearchState>` at **line 105-132** wraps the results. `AsyncLoader`
(`lib/components/async_loader.dart:24-26`) renders `WandererError(err: ...)` on `hasError`, and
`WandererError` (`lib/components/base/wanderer_error.dart:31`) shows `l10n.something_went_wrong`
plus the raw `err.toString()` — that is exactly the "generic Something went wrong" from CONTEXT.
The takeover goes inside the `Expanded`/`RefreshIndicator` at lines 102-134 (branch on
`onlineStatusProvider` before reaching `AsyncLoader`). The search field (66-100) and
`ListQuickFilterBar` (101) sit above and can stay. Retry: `ref.invalidate(listSearchProvider)`
(same call already used by the pull-to-refresh at line 104).

**Reusable offline empty-state precedent — use this, don't invent a new one.**
`lib/routes/settings_offline_regions_screen.dart:427-462` defines `_buildEmptyState({title, body,
icon})` and line 250-258 uses it with `FontAwesomeIcons.linkSlash` (the source comment notes
`wifiSlash` is Pro-only and unavailable in `font_awesome_flutter`'s free set — **do not try to use
`wifiSlash`**). Existing l10n keys to model copy on: `regions_offline_unavailable_title`
("Can't load regions"), `regions_offline_unavailable_body`, `regions_retry` ("Retry"), `offline`
("Offline"). Consider extracting `_buildEmptyState` into a shared
`lib/components/base/wanderer_offline_state.dart` with an added `onRetry` callback.

## Finding 4 — Bottom nav avatar + profile screen

**Bottom nav:** `app/lib/components/base/wanderer_layout.dart`. The whole bar is a `BottomAppBar`
(line 60-114). Avatar is at **lines 94-105**:

```dart
_NavItem(
  icon: CircleAvatar(
    radius: 12,
    backgroundImage: NetworkImage(
      user?.getFileUrl(user.serverUrl, user.avatar) ??
          "https://api.dicebear.com/7.x/initials/png?seed=${user?.preferredUsername}&...",
    ),
    onBackgroundImageError: (_, _) => FaIcon(FontAwesomeIcons.user),   // line 104
  ),
  ...
```

`user` comes from `ref.watch(authProvider).value` (line 16) — a `UserEntity` read straight out of
ObjectBox, so the *metadata* already survives offline (`Auth.build()` at
`lib/provider/auth_provider.dart:39-80` deliberately falls back to the cached entity on timeout or
non-auth error). Only the **image bytes** are missing.

**Two real bugs to fix here:**
1. **`onBackgroundImageError` at line 104 is a no-op.** Its type is `ImageErrorListener`
   (`void Function(Object, StackTrace?)`); returning a `FaIcon` from the arrow body discards it.
   The avatar renders as a blank grey circle on failure, not a user icon. Same discard pattern
   exists at `lib/routes/profile_screen.dart:132` and `:236` (`(e, _) {}` — at least honest there).
   Fix: render a `child:` fallback / branch the widget instead of relying on the callback.
2. **The dicebear fallback URL is itself a network fetch** — offline it fails too, so the "no
   avatar" path is equally blank. Needs a local asset or `FaIcon` fallback when offline.

**No disk image cache exists.** Verified: `cached_network_image` is not in `pubspec.yaml`; there
are ~20 raw `NetworkImage(` sites across `lib/`. The only local-image precedent is
`lib/components/trail/trail_card.dart:49` and `lib/components/trail/trail_list_item.dart:39`
using `FileImage(file)` for photos already downloaded by
`lib/services/trail_download_service.dart:221-247` (dio download → `<appDoc>/.../photos/`).

Recommended approach (no new dependency, mirrors the existing precedent):
- On a successful `Auth._updateUserEntity` (`lib/provider/auth_provider.dart:226-248`), download
  the avatar once to `<appDocDir>/avatars/<userId>_<filename>` and store the path.
- Storage options: (a) add a nullable `avatarLocalPath` field to `UserEntity`
  (`lib/entities/user_entity.dart`) — **requires an ObjectBox schema change + `build_runner`**, or
  (b) derive the path deterministically from `user.id` + `user.avatar` and just check
  `File(path).existsSync()` — **no schema change**, preferred for a quick task.
- Render `FileImage(file)` when the cached file exists, `NetworkImage` when online, `FaIcon`
  otherwise.

**Profile screen:** `app/lib/routes/profile_screen.dart` (610 lines).
- **Line 74-80** is the spinner-then-error the CONTEXT describes:
  ```dart
  body: actorAsync.when(
    data: (actor) => _buildProfile(actor),
    loading: () => const Center(child: CircularProgressIndicator()),   // :77
    error: (err, stack) => WandererError(err: err, stack: stack),      // :78
  ),
  ```
- `actorAsync` = `ref.watch(ownProfileProvider)` for the own profile (line 72).
  `OwnProfile.build()` (`lib/provider/profile/profile_provider.dart:35-42`) does
  `api.get('/profile/${user.preferredUsername}')` with **no cached fallback** — that is the error.
- **Settings entry point is `_actionButtons` at lines 190-206** — the gear `IconButton` pushing
  `/settings` lives in the `SliverAppBar.actions` (line 159), i.e. **inside `_buildProfile`**.
  That is precisely why settings become unreachable offline: the error branch at line 78 replaces
  the entire scaffold body including the app bar. Rendering a cached profile fixes settings access
  automatically. `/settings` itself is a **top-level GoRoute outside the shell**
  (`lib/provider/router_provider.dart:194`) and `SettingsScreen`
  (`lib/routes/settings_screen.dart`) is a plain `ListView` of `ListTile`s with zero network calls
  — so once reachable, settings already work offline.

**Cached-actor path already exists in the schema.** `lib/entities/actor_entity.dart` has
`ActorEntity.fromModel(Actor)` **and** an `ActorEntityMapping.toModel()` extension (lines 61-110)
covering every `Actor` field. It is currently only persisted as `TrailEntity.author`
(`lib/entities/trail_entity.dart:60,121`) — never for the current user. Two options:
- (a) Persist the own-profile `Actor` as an `ActorEntity` on each successful `OwnProfile` fetch and
  fall back to `toModel()` offline. Cleanest, gives a full `Actor` and reuses existing code.
- (b) Synthesize a minimal `Actor` from `UserEntity` — but `Actor` requires `iri`, `inbox`,
  `publicKey`, `lastFetched`, `user` (all non-nullable), and `UserEntity` has only `iri`/`actorId`,
  so several fields would need placeholder strings. Messier.
Recommend (a).

Sub-sections needing "You are offline" empty states rather than errors:
`_ListsPreview` → `AsyncLoader<ProfileListsState>` at **line 325**; `_FeedSection` →
`AsyncLoader<ProfileFeedState>` at **line 379** (+ inline spinner at 403); `_CountsRow` →
`profileCountsProvider` (`lib/provider/profile/profile_counts_provider.dart:12`); `_StatsRow`
follow button → `followAsync.when` at **line 584** (disable when offline).

## Common Pitfalls

### Pitfall 1: Mutual synchronous provider initialization
**What goes wrong:** `OnlineStatus.build()` reading `apiProvider` while `Api.build()` resolves
`onlineStatusProvider.notifier` → re-entrant init.
**Avoid:** `build() => true` with no reads; resolve the notifier lazily inside the interceptor
closures. `refresh()` may read `apiProvider` freely because it runs post-build.

### Pitfall 2: "Tried to modify a provider while the widget tree was building"
**Verified:** `flutter_riverpod-3.3.1/lib/src/core/provider_scope.dart:322-345` — a
**debug-mode-only** guard that fires when provider state is written during `build`/`initState`/
`didChangeDependencies`. Dio interceptor callbacks run in the async request pipeline, so they are
safe. The launch seed is safe too because `refresh()` writes state only after an `await`. If a
synchronous write ever becomes necessary from a lifecycle method, defer it with
`Future.microtask` or `addPostFrameCallback`.

### Pitfall 3: Notification storms on every response
`markOnline()`/`markOffline()` guard on the current value, so a no-op write is skipped. (Riverpod's
default `updateShouldNotify` is `!identical(prev, next)` and `identical(true, true)` is `true`, so
even an unguarded write wouldn't notify — but the explicit guard documents intent.)

### Pitfall 4: keepAlive providers cache their offline error
`mapStyleSourcesProvider`, `mapStyleJsonProvider`, `ownProfileProvider`, `mapTrailSearchProvider`,
`mapClusterSearchProvider` are all `@Riverpod(keepAlive: true)`. Coming back online does **not**
re-run them. Every "Try again" CTA must `ref.invalidate(...)` the specific providers the screen
depends on, not just flip `onlineStatusProvider`.

### Pitfall 5: `/health` returning 503 conflicts with the interceptor rule
A 503 (proxy up, PocketBase down) is `badResponse` ⇒ the interceptor marks **online**, while
`isBackendReachable` returns `false` ⇒ `refresh()` then writes `false`. Correct final state
because `refresh()` assigns last — but the **next** unrelated API call's `badResponse` would flip
it back to online. This follows the CONTEXT decision literally; flagged as an open question below.

### Pitfall 6: Not every Dio in the app is the shared client
`lib/provider/welcome/server_selection_provider.dart:18` constructs its own `Dio()` for the
pre-login server probe, and `lib/services/tile_proxy_server.dart` serves tiles over loopback.
Neither goes through the interceptor — correct, but means online status is not updated during
server selection / tile fetches. Do not "fix" this.

### Pitfall 7: `Api` rebuilds drop the interceptor
`Api.build()` does `ref.watch(cookieJarProvider)` (`api_provider.dart:12`), so a cookie-jar change
rebuilds `Dio` and re-runs the interceptor registration — fine. In practice `cookieJarProvider` is
`overrideWithValue`'d in `main.dart:59` so it never changes.

### Pitfall 8: Codegen must be re-run
New provider ⇒ `online_status_provider.g.dart` must be generated. If a `UserEntity` field is
added, `objectbox.g.dart` + `lib/objectbox-model.json` change too.
```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```
l10n: new keys go in `lib/i18n/app_en.arb` (template, per `app/l10n.yaml`); there are 14 other
`.arb` locale files — `flutter gen-l10n` runs automatically via `generate: true` in `pubspec.yaml`.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| dio | interceptor | ✓ | 5.9.2 (locked) | — |
| flutter_riverpod | provider | ✓ | 3.3.1 (locked) | — |
| riverpod (core) | Ref semantics | ✓ | 3.2.1 (transitive) | — |
| riverpod_annotation / riverpod_generator | codegen | ✓ | 4.0.2 / 4.0.3 | — |
| build_runner | codegen | ✓ | 2.13.1 | — |
| objectbox / objectbox_generator | UserEntity/ActorEntity | ✓ | 5.3.1 | — |
| font_awesome_flutter | `linkSlash` icon | ✓ | 11.0.0 | — |
| skeletonizer | loading shimmer | ✓ | 2.1.3 | — |
| slopcheck | package audit | ✓ | installed | — |
| ctx7 CLI / Context7 MCP | docs lookup | ✗ | — | Read installed package source (used) |

**Missing dependencies with no fallback:** none.

## Package Legitimacy Audit

**No new packages are recommended.** Every mechanism this task needs already exists in
`app/pubspec.yaml`.

| Package | Registry | Disposition |
|---------|----------|-------------|
| (none) | — | No installs required |

*Alternative considered:* `cached_network_image` (pub.dev) would give disk-backed avatar caching
across all ~20 `NetworkImage(` sites. **Not verified against pub.dev in this session** — treat as
`[ASSUMED]`. If the planner chooses it over the FileImage approach, gate it behind a
`checkpoint:human-verify` and confirm on pub.dev first. `slopcheck` targets npm/PyPI, not pub.dev,
so it cannot vet Dart packages here.

## Existing Tests

`grep -rn "isBackendReachable" test/` → **no matches.** No test asserts on the probe being called,
so the signature change is safe. Nothing in `test/` overrides `apiProvider` either
(`test/routes/settings_account_screen_test.dart:88` has a `// TODO:` noting the fixture is missing).

Relevant existing test dirs the planner may extend: `test/provider/` (has
`auth_provider_refresh_test.dart` — the closest analogue for a provider test),
`test/util/`, `test/routes/` (widget tests for settings screens).

Files currently modified in the working tree (`route_anchor_provider.dart`,
`planned_gpx_provider.dart`, `route_planner_screen.dart`, `trail_source_select_screen.dart`,
`route_planner_handoff_util.dart`, `route_travel_bucket.dart`,
`route_planner_handoff_util_test.dart`) are **route-planner / Valhalla-costing work, unrelated to
offline handling** — except `trail_source_select_screen.dart`, which contains migration call site
#3 at line 99. Expect to edit a file that already has uncommitted changes.

## Security Domain

`security_enforcement: true`, ASVS level 1. This task adds no auth, crypto, or new input surface.

| ASVS Category | Applies | Note |
|---------------|---------|------|
| V2 Authentication | no | No auth logic changes; `Auth` notifier untouched apart from optional avatar caching. |
| V3 Session Management | no | Cookie jar unchanged. |
| V4 Access Control | no | — |
| V5 Input Validation | marginal | The cached avatar filename comes from the server; if writing it to disk, sanitise the path component (`p.basename`) so a crafted filename cannot traverse out of the avatars dir. |
| V6 Cryptography | no | — |

One concrete item: **do not log or surface `err.toString()` from `DioException` in the offline UI**
— the current `WandererError` (`wanderer_error.dart:39-45`) prints the raw exception, which can
include the full request URL. The offline takeover should show static copy instead. This is a
side benefit of the change.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `cached_network_image` exists on pub.dev at a current version | Package Legitimacy Audit | Only matters if the planner picks it over FileImage; verify on pub.dev first |
| A2 | The SvelteKit proxy `/health` endpoint returns 503 (not a connection error) when PocketBase is down | Pitfall 5 | Only affects the 503 edge case; the doc comment in `connectivity_util.dart:8-9` asserts this |
| A3 | Deterministic avatar paths (no `UserEntity` schema change) are sufficient | Finding 4 | If the avatar filename can change without the user id changing, stale files accumulate; mitigate with a cleanup on write |

## Open Questions

1. **503-from-`/health` semantics.** Per the CONTEXT decision, any response that reached the server
   marks online — so a backend whose database is down reads as "online" after the next API call,
   even though nothing works. Recommendation: ship the literal CONTEXT rule (simplest, matches the
   decision), and treat "server up, DB down" as a separate future concern.
2. **Avatar caching mechanism.** FileImage + deterministic path (no new dep, no schema change) vs.
   `cached_network_image` (broader benefit, new dep). Recommendation: FileImage for this quick task.
3. **Own-profile Actor cache.** Persist `ActorEntity` for the current user (recommended) vs.
   synthesize a partial `Actor` from `UserEntity`. Recommendation: persist `ActorEntity` — the
   `fromModel`/`toModel` round-trip already exists and is fully field-complete.

## Sources

### Primary (HIGH confidence — read directly)
- `~/.pub-cache/hosted/pub.dev/dio-5.9.2/lib/src/dio_exception.dart:15-41` — `DioExceptionType`
- `~/.pub-cache/hosted/pub.dev/dio-5.9.2/lib/src/adapters/io_adapter.dart:113,131` — SocketException → connectionError
- `~/.pub-cache/hosted/pub.dev/dio-5.9.2/lib/src/interceptor.dart:209-230` — callback signatures
- `~/.pub-cache/hosted/pub.dev/riverpod-3.2.1/lib/src/core/ref.dart:537-546,652-661` — read vs watch
- `~/.pub-cache/hosted/pub.dev/flutter_riverpod-3.3.1/lib/src/core/provider_scope.dart:322-345` — build-phase guard
- App source: `lib/provider/api_provider.dart`, `lib/util/connectivity_util.dart`,
  `lib/main.dart:32-64,78-115,233-291`, `lib/routes/map_screen.dart`, `lib/routes/list_screen.dart`,
  `lib/routes/profile_screen.dart`, `lib/routes/trail_source_select_screen.dart:88-127`,
  `lib/components/base/wanderer_layout.dart`, `lib/components/base/trail_collection_map.dart`,
  `lib/components/async_loader.dart`, `lib/components/base/wanderer_error.dart`,
  `lib/provider/auth_provider.dart`, `lib/provider/profile/profile_provider.dart`,
  `lib/provider/map_style_sources_provider.dart`, `lib/entities/user_entity.dart`,
  `lib/entities/actor_entity.dart`, `lib/routes/settings_offline_regions_screen.dart:225-462`,
  `lib/routes/settings_screen.dart`, `lib/provider/router_provider.dart:122-260`,
  `app/pubspec.yaml`, `app/pubspec.lock`, `app/l10n.yaml`, `lib/i18n/app_en.arb`

### Not used
- Context7 MCP / `ctx7` CLI — unavailable in this environment. Compensated by reading the exact
  installed package sources, which is strictly more authoritative for version-pinned behaviour.

## Metadata

**Confidence breakdown:**
- dio condition: HIGH — read the pinned 5.9.2 enum and io adapter directly
- Riverpod wiring / no-cycle claim: HIGH — read `Ref.read` and `Ref.watch` implementations
- Screen structure & line references: HIGH — read every referenced file
- Avatar caching recommendation: MEDIUM — no in-repo precedent for avatar caching specifically;
  the FileImage/download pattern is extrapolated from `trail_download_service.dart`

**Research date:** 2026-07-29
**Valid until:** 2026-08-28 (versions are lock-pinned; findings only go stale on a dep upgrade)
