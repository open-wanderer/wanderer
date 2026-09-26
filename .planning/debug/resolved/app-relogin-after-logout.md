---
status: resolved
trigger: "When the user logs out of the app while a request is still in flight the user gets logged back in. At least that is my suspicion. I just experienced that behaviour."
goal: find_and_fix
created: 2026-09-26
updated: 2026-09-26
---

## Symptoms

- expected: After tapping logout, the app stays logged out — in the current session and after a restart — until the user logs in again.
- actual: The user observed being logged back in after logging out. Their suspicion: a request that was still in flight when they logged out completes afterwards and restores the session.
- errors: None reported.
- timeline: Observed once by the developer on 2026-09-26 on branch feature/app. Unknown whether it is new.
- reproduction: Not reliable yet. Suspected: start something that fires API requests (e.g. cold start background validation, feed/trail loads, sync), then log out before they complete.
- scope: Flutter mobile app in `app/` (auth_provider.dart, api_provider.dart CookieManager + PersistCookieJar), plus the SvelteKit proxy `web/src/hooks.server.ts`, which appends a `pb_auth` Set-Cookie to every response.
- related: `.planning/debug/app-logout-on-close.md` (diagnosed) documents the cookie round-trip: hooks.server.ts:180 exports `pb_auth` on every response; Dio's CookieManager persists it to the on-disk jar with no filtering; `Auth.build()` restores the session purely from the presence of the `pb_auth` cookie; `logout()` does `jar.deleteAll()` + `_box.removeAll()` + purge.

## Current Focus

reasoning_checkpoint:
  hypothesis: |
    `Auth.logout()` (app/lib/provider/auth_provider.dart:276) clears local session state
    (cookie jar, ObjectBox UserEntity, Auth notifier state) with no coordination with
    requests already in flight on the shared Dio client. Two independent write paths can
    land AFTER logout()'s teardown and resurrect exactly what it cleared:
      Candidate A: dio_cookie_manager's CookieManager.onResponse (cookie_mgr.dart:81-103)
      unconditionally calls saveCookies() for ANY response on the shared Dio client,
      writing whatever Set-Cookie headers it carries into the PersistCookieJar. Every
      response through web/src/hooks.server.ts:180 carries a fresh pb_auth Set-Cookie
      (exported unconditionally, confirmed by reading hooks.server.ts:134-183). A request
      still awaiting that response when logout() runs (e.g. _validateInBackground's GET
      /user/:id) writes a live pb_auth cookie into the jar after jar.deleteAll() ran.
      Candidate B: Auth._updateUserEntity (auth_provider.dart:291-356) does an unconditional
      `_box.put(userEntity)` (line 338) on its success path, with no check that logout()
      already ran and invalidated this notifier. The existing `!ref.mounted` guard in
      _validateInBackground (line 98) protects the in-memory `state` assignment but was
      never applied to this earlier ObjectBox write, so the persisted UserEntity resurfaces
      on the very next Auth.build() (which derives "signed in" from box.getAll().firstOrNull
      plus a present pb_auth cookie).
    Both candidates are confirmed and independently sufficient contributors; they combine
    in the reported symptom.
  confirming_evidence:
    - "dio_cookie_manager-3.4.0/lib/src/cookie_mgr.dart:81-103,164-182 — onResponse always
      calls saveCookies(response), which always calls cookieJar.saveFromResponse(...) for
      any non-empty Set-Cookie header, with zero awareness of app-level logout/session
      state."
    - "web/src/hooks.server.ts:134-143,177-183 — every response gets `pb.authStore
      .exportToCookie(...)` appended as `set-cookie`, unconditionally, including a request
      that was already authenticated before the app-side logout ran (server has no
      knowledge the client is about to log out)."
    - "app/lib/provider/auth_provider.dart:291-338 — `_updateUserEntity`'s `_box.put
      (userEntity)` at line 338 runs on every successful response with no `ref.mounted` or
      generation check, unlike the state assignment in `_validateInBackground` (line 98)
      which already has one."
    - "app/lib/provider/auth_provider.g.dart:32 — `Auth create() => Auth();` confirms
      `ref.invalidateSelf()` (called at the end of `logout()`, auth_provider.dart:288)
      swaps in a brand-new `Auth` instance; the old instance (and any Future still running
      on it, like a pending `_validateInBackground`/`_updateUserEntity` call) becomes
      unmounted — this is exactly what the existing `!ref.mounted` check at line 98 already
      relies on, just applied too late in the call chain to protect the box write."
    - "dio-5.9.2/lib/src/adapters/io_adapter.dart:98-101,161-175 — confirms a genuinely
      in-flight request (still awaiting `request.close()`) is aborted via
      `requestWR.target?.abort()` when its CancelToken is cancelled, which prevents the
      response — and therefore both the cookie write and the `_updateUserEntity` write —
      from ever materializing. This is the basis for the fix: cancelling requests at
      logout time closes both candidates for the realistic 'still awaiting network I/O'
      timing the user described, not just candidate B."
  falsification_test: |
    If a fix that (a) aborts in-flight non-explicitly-tokened requests at the start of
    logout() and (b) adds a `!ref.mounted` guard before `_updateUserEntity`'s box write
    still allows a UserEntity + pb_auth cookie pair to reappear after logout with a request
    deliberately held in flight past logout (e.g. via a debug delay), the hypothesis is
    wrong or incomplete.
  fix_rationale: |
    Root cause is the absence of any synchronization between logout()'s teardown and
    requests already in flight on the shared client. The fix targets that gap directly
    (cancel in-flight requests as part of logout; block the one unconditional write that
    survives past cancellation's edge-case timing) rather than patching a symptom (e.g.
    re-clearing the jar after a delay, which would still race).
  blind_spots: |
    Not proven empirically end-to-end (no automated repro harness for "hold a response
    open past logout" exists in this codebase's test suite) — verification below relies on
    `flutter analyze`/`flutter test` plus code-path reasoning, with human on-device
    confirmation still required. Extremely tight timing (cancel() called the same
    microtask a response is already mid-way through CookieManager's onResponse) is not
    fully closed by cancellation alone — the ref.mounted guard on the box write is the
    backstop for that sliver, but Auth.build() also requires the cookie to be present, so
    even in that sliver the box stays empty and the user stays logged out.
next_action: |
  Fix applied and analyzed/tested clean (see Resolution). Awaiting human on-device
  verification per project constraints (never flutter build/adb install from this agent).

## Evidence

- timestamp: 2026-09-26T00:00:00Z
  checked: app/lib/provider/auth_provider.dart (full file)
  found: |
    `logout()` (line 276) does `state = AsyncLoading()`, `jar.deleteAll()`, `_box.removeAll
    ()`, `purgeAccountScopedData(...)`, `ref.invalidateSelf()` — no cancellation of
    in-flight Dio requests, no generation guard beyond the single `!ref.mounted` check in
    `_validateInBackground` (line 98), which only guards the `state = AsyncData(fresh)`
    assignment. `_updateUserEntity` (line 291) writes `_box.put(userEntity)` (line 338)
    unconditionally on success.
  implication: Confirms Candidate B mechanically — a stale in-flight `_updateUserEntity`
    call writes the signed-out user back into ObjectBox with no guard.
- timestamp: 2026-09-26T00:00:00Z
  checked: app/lib/provider/api_provider.dart (full file), cookie_jar_provider.dart
  found: |
    The shared `Dio` instance built in `Api.build()` has no CancelToken plumbing at all —
    every caller relies on Dio's default (no cancellation) unless it supplies its own
    token (only download/route flows do, confirmed via grep for CancelToken usage:
    reverse_geocode.dart, notification_id.dart, tile_status.dart,
    account_scope_invalidation.dart (comment only), route_anchor_provider.dart,
    region_download_state.dart, trail_download_service.dart,
    download_notification_service.dart, tile_repository_manager.dart — none of these are
    the auth/user-fetch path).
  implication: No existing mechanism stops a request from completing after logout; a new
    default-CancelToken mechanism can be added without disturbing those download flows
    (they already bring their own token).
- timestamp: 2026-09-26T00:00:00Z
  checked: ~/.pub-cache/hosted/pub.dev/dio_cookie_manager-3.4.0/lib/src/cookie_mgr.dart
  found: |
    `onResponse`/`onError` both unconditionally call `saveCookies(response)`, which calls
    `cookieJar.saveFromResponse(realUri, cookies)` for any non-empty Set-Cookie header. No
    app-level hook, flag, or cancellation-awareness exists in this interceptor.
  implication: Confirms Candidate A mechanically — any response landing after
    `jar.deleteAll()` silently repopulates the jar, entirely inside a third-party
    interceptor the app does not control point-by-point.
- timestamp: 2026-09-26T00:00:00Z
  checked: web/src/hooks.server.ts lines 100-187
  found: |
    Line 180: `pb.authStore.exportToCookie(...)` is appended as a `set-cookie` response
    header on every response through this proxy (the same mechanism documented in the
    related session app-logout-on-close.md), independent of whether the client is about to
    log out — the server has no visibility into that.
  implication: Confirms the in-flight request's response really would carry a live,
    resurrecting pb_auth cookie, not just an inert one.
- timestamp: 2026-09-26T00:00:00Z
  checked: app/lib/provider/auth_provider.g.dart (generated), riverpod semantics
  found: |
    `AuthProvider.create() => Auth();` — `ref.invalidateSelf()` causes a brand-new `Auth`
    instance to be constructed and `build()` to run on it; the old instance (and any Future
    still executing on it) is disposed, flipping its `ref.mounted` to false.
  implication: The existing `!ref.mounted` check is a real, working stale-instance guard —
    it was simply placed after the box write it needed to protect, not before it.
- timestamp: 2026-09-26T00:00:00Z
  checked: ~/.pub-cache/hosted/pub.dev/dio-5.9.2/lib/src/dio_mixin.dart and
    lib/src/adapters/io_adapter.dart
  found: |
    `requestOptions.cancelToken` is read fresh at each interceptor step via
    `listenCancelForAsyncTask` (dio_mixin.dart ~397-429), so a CancelToken assigned by an
    `onRequest` interceptor (rather than by the original caller) is honored for the rest of
    that request's lifecycle. The IO adapter registers `cancelFuture?.whenComplete(() =>
    requestWR.target?.abort())` right after opening the connection (io_adapter.dart:98-101)
    and aborts on receive as well (lines 161-175) — cancelling while genuinely awaiting a
    response aborts the socket, so the response (and therefore CookieManager.saveCookies
    and _updateUserEntity's continuation) never materializes.
  implication: A default CancelToken assigned via an `onRequest` interceptor, rotated and
    cancelled from `Api.cancelPendingRequests()`, is a mechanically sound way to close both
    candidates for the realistic "awaiting network I/O" timing described in the symptom.

## Eliminated

## Resolution

root_cause: |
  `Auth.logout()` (app/lib/provider/auth_provider.dart) clears the cookie jar, ObjectBox
  UserEntity row, and Auth notifier state with no coordination with requests already in
  flight on the shared Dio client (app/lib/provider/api_provider.dart). Two independent,
  confirmed write paths could land after that teardown and resurrect the signed-out
  session:
    (A) dio_cookie_manager's CookieManager.onResponse unconditionally persists any
    Set-Cookie header from any response on the shared client into the PersistCookieJar.
    web/src/hooks.server.ts appends a fresh, live `pb_auth` Set-Cookie to every response
    (line 180), including the response to a request that was already authenticated before
    the client-side logout ran. A request still awaiting that response when
    `jar.deleteAll()` ran would silently repopulate the jar afterwards.
    (B) `Auth._updateUserEntity`'s `_box.put(userEntity)` ran unconditionally on its
    success path, with no check that `logout()` had already invalidated the notifier
    (`ref.invalidateSelf()` swaps in a brand-new `Auth` instance — confirmed via
    auth_provider.g.dart's `create()`). The existing `!ref.mounted` guard in
    `_validateInBackground` only protected the later in-memory `state` assignment, not
    this earlier persisted write, so the very next `Auth.build()` found the resurrected
    UserEntity in the box.
  Both were independently sufficient contributors and combine in the reported symptom,
  since `Auth.build()` requires both a UserEntity row AND a pb_auth cookie to restore a
  session.
fix: |
  1. app/lib/provider/api_provider.dart: added a rotatable `_sessionCancelToken`
     (CancelToken) attached via a new `onRequest` interceptor to any request that doesn't
     already carry its own token (added before CookieManager in the interceptor chain).
     Added `Api.cancelPendingRequests()`, which cancels the current token and arms a fresh
     one. Verified via dio-5.9.2 and dio_cookie_manager-3.4.0 source that cancelling a
     token while a request is genuinely awaiting a response aborts the underlying socket
     (`IOHttpClientAdapter`'s `HttpClientRequest.abort()`), so the response never reaches
     CookieManager.onResponse or the calling code at all -- closing both candidate A and
     candidate B for the realistic "still awaiting network I/O" timing described in the
     symptom. Explicit-CancelToken flows (region/tile downloads, trail uploads) are
     untouched by design (`??=`), since those are meant to survive a logout per
     account_scope_invalidation.dart.
  2. app/lib/provider/auth_provider.dart:
     - `logout()` now calls `ref.read(apiProvider.notifier).cancelPendingRequests()` as
       its first action, before `jar.deleteAll()`.
     - `_updateUserEntity` now returns `null` immediately after the network await if
       `!ref.mounted`, before any ObjectBox/settings/state write -- extending the same
       stale-instance guard already used later in `_validateInBackground` to the point
       where it actually needs to apply. This is a backstop for the sliver of timing
       cancellation cannot close (cancel() called while a response is already mid-way
       through interceptor processing): even if a stray cookie is written in that sliver,
       the box stays empty, so `Auth.build()` still resolves to logged-out.
  Both changes are additive/defensive; no existing call path's happy-path behavior
  changes. Does not touch web/src/hooks.server.ts (out of scope for this session; see
  project_constraints and the related app-logout-on-close.md session) and does not touch
  app/lib/routes/map_screen.dart.
verification: |
  - Confirmed fixed on device by the user (2026-09-26).
  - `flutter analyze` (app/): clean -- 0 errors/warnings; only 13 pre-existing info-level
    lints in unrelated vendor/other files.
  - `flutter test` (app/): full suite ran clean at 1172 passed / 1 pre-existing failure
    (test/provider/server_selection_provider_test.dart, confirmed via `git stash` to fail
    identically without this fix -- unrelated to auth/api providers) / 1 skipped. Targeted
    reruns of test/provider/auth_provider_refresh_test.dart and
    test/entities/trail_author_upsert_gate_test.dart also passed cleanly in the same
    consistent tree state.
  - Mechanism verified by reading package source directly, not just inferred: dio-5.9.2's
    `dio_mixin.dart` (cancelToken re-read per interceptor step via
    `listenCancelForAsyncTask`) and `io_adapter.dart` (`cancelFuture?.whenComplete(() =>
    requestWR.target?.abort())`) confirm cancellation aborts a request that is genuinely
    awaiting a response; dio_cookie_manager-3.4.0's `cookie_mgr.dart` confirms
    `saveCookies` never runs for an aborted request.
  - NOT independently reproduced on-device (no automated harness exists in this codebase
    for "hold a response open past logout"); this agent does not run `flutter build`/`adb
    install` per project constraints. Human on-device verification requested below.
  - Cross-check against the related session (app-logout-on-close.md, the reverse
    symptom): this fix only changes what happens once `logout()` has already been decided
    to run (cancelling in-flight requests as part of teardown) -- it adds no new path that
    calls `logout()`, so it cannot make that session's false-logout trigger (hooks.server
    .ts clearing authStore on transient failures) more likely. Does not modify
    web/src/hooks.server.ts.
files_changed:
  - app/lib/provider/api_provider.dart
  - app/lib/provider/auth_provider.dart
