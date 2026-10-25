---
status: diagnosed
trigger: "People are complaining that they get logged out of the app often. As in, when the app closes. Can you find a reason for that? Do not fix anything. Come back with findings first"
goal: find_root_cause_only
created: 2026-09-26
updated: 2026-09-26T00:40:00Z
---

## Symptoms

- expected: A logged-in user stays logged in across app restarts (app closed / killed by OS / relaunched).
- actual: Users report being logged out "often", apparently when the app closes and is reopened.
- errors: None reported (user-relayed tester complaints).
- timeline: Unknown; reported by beta testers of the Flutter app (branch feature/app).
- reproduction: Not reproduced yet. Log in, close the app (swipe away / OS kills it), reopen → sometimes shows logged-out state.
- scope: Flutter mobile app in `app/` (auth via PocketBase SDK, session persistence).
- constraint: DIAGNOSE ONLY — do not change any code. Report findings first.
- related: `.planning/debug/app-offline-after-login.md` (awaiting_human_verify) touches the post-login state path; may share code with auth restore.

## Current Focus

reasoning_checkpoint:
  hypothesis: |
    web/src/hooks.server.ts:134-143 attempts `pb.collection('users').authRefresh()`
    on EVERY request that carries a currently-valid token, and its catch block
    unconditionally does `pb.authStore.clear()` on ANY failure of that call — not
    just a genuinely revoked/expired token, but also a transient network/backend
    hiccup between the SvelteKit server and PocketBase. Because
    `pb.authStore.exportToCookie(...)` is then sent back on literally every
    response (hooks.server.ts:180, unconditional, outside the try/catch), a
    single transient refresh failure sends the Flutter app a `pb_auth` cookie
    that deletes the previously-valid session cookie (expires 1970-01-01,
    confirmed via the PocketBase JS SDK's `exportToCookie`/`getTokenPayload`
    source). The Flutter app's Dio client auto-persists that Set-Cookie via
    `dio_cookie_manager` into the on-disk `PersistCookieJar`
    (api_provider.dart:61, main.dart:56-59), and `cookie_jar` package's own
    RFC6265 semantics permanently remove the prior valid cookie from the file on
    disk. On the next cold start, `Auth.build()` (auth_provider.dart:42-67) finds
    no `pb_auth` cookie and reports the user as logged out, even though nothing
    was ever wrong with their actual PocketBase session.

    This is aggravated by a same-request effect: because the authStore is
    cleared BEFORE `event.locals.user` is set (hooks.server.ts:148-164), the very
    request that hit the transient refresh failure is itself downgraded to
    anonymous. `Auth._validateInBackground` (auth_provider.dart:85-109), which
    runs unconditionally on every cold start via `Auth.build()` (line 65), calls
    `_updateUserEntity` → `GET /api/v1/user/:id`
    (web/src/routes/api/v1/user/[id]/+server.ts:36-44 →
    web/src/lib/util/api_util.ts's `show()`), which now runs unauthenticated
    against the `users` collection. PocketBase's own view-rule behavior returns
    403/404 for a record an anonymous caller cannot see, and
    `_isAuthError` (auth_provider.dart:133-139) treats 401/403/404 as proof of
    an invalid session, triggering a FULL local `logout()` — clearing the cookie
    jar, the cached `UserEntity`, and purging account-scoped local data
    (auth_provider.dart:276-289) — from a single transient upstream hiccup that
    had nothing to do with the user's actual credentials.

    "Happens when the app closes and reopens" fits because `_validateInBackground`
    fires on every single cold start (the highest-frequency trigger of this exact
    vulnerable request), while the underlying wipe can equally happen mid-session
    on any other authenticated request — cold start is just where the user
    reliably notices it as "logged out."
  confirming_evidence:
    - "web/src/hooks.server.ts:134-143 — `if (pb.authStore.isValid) { await pb.collection('users').authRefresh(...) }` wrapped in a bare `try/catch(_) { pb.authStore.clear(); ... }` with no differentiation between a real 401 (revoked/expired token) and a network/5xx/timeout failure reaching PocketBase."
    - "web/src/hooks.server.ts:148-164 — `event.locals.user = pb.authStore.record` runs AFTER the refresh attempt, so a request that arrived with a perfectly valid cookie is downgraded to anonymous for its own remaining lifecycle once refresh fails."
    - "web/src/hooks.server.ts:180 — `pb.authStore.exportToCookie(...)` + `response.headers.append('set-cookie', pbCookie)` run unconditionally on every response, regardless of whether the store was just cleared."
    - "web/node_modules/pocketbase/dist/pocketbase.cjs.js — `exportToCookie`: `s.expires = n?.exp ? new Date(1e3*n.exp) : new Date(\"1970-01-01\")`; `getTokenPayload(\"\")` returns `{}` (no `exp`) after `clear()` sets `baseToken=\"\"`. Confirms a cleared authStore serializes to an already-expired (deleting) Set-Cookie header."
    - "app/lib/provider/api_provider.dart:55-61 — shared Dio client base URL is `$serverUrl/api/v1` (the SvelteKit proxy, not PocketBase directly) and has `dio.interceptors.add(CookieManager(cookieJar))`, so every Set-Cookie header from hooks.server.ts is auto-persisted with no filtering."
    - "app/lib/main.dart:50-59 — `PersistCookieJar(storage: FileStorage(cookiePath), ignoreExpires: false)`, an on-disk jar in the app documents directory."
    - "~/.pub-cache/hosted/pub.dev/cookie_jar-4.0.9/lib/src/jar/default.dart:161-165 (`saveFromResponse`) — the incoming expired `pb_auth` cookie overwrites the existing map entry, then `if (_isExpired(...)) map.remove(cookie.name);` deletes it immediately in memory."
    - "~/.pub-cache/hosted/pub.dev/cookie_jar-4.0.9/lib/src/jar/persist.dart:141-154 (`_filterPathEntries`, used by `_save`) — `if (cookie.isExpired()) continue;` means the on-disk file is rewritten without `pb_auth`, making the deletion durable across restarts."
    - "app/lib/provider/auth_provider.dart:42-67 (`Auth.build`) — cold-start restore is driven entirely by presence of the `pb_auth` cookie (`cookies.where((c) => c.name == 'pb_auth').firstOrNull`); if absent, returns null (shown as logged out) even though the ObjectBox `UserEntity` row (the actual cached account) is still present at line 47."
    - "app/lib/provider/auth_provider.dart:65,85-109 (`_validateInBackground`) — runs unconditionally on every `Auth.build()`, i.e. every cold start, calling `_updateUserEntity(cached.id)`."
    - "app/lib/provider/auth_provider.dart:133-139 (`_isAuthError`) — treats DioException status 401, 403, OR 404 as proof of an invalid session and calls `logout()`."
    - "web/src/routes/api/v1/user/[id]/+server.ts:36-44 + web/src/lib/util/api_util.ts `show()` (`event.locals.pb.collection('users').getOne(id, ...)`) — the exact endpoint `_updateUserEntity` calls; PocketBase's own view-rule-mismatch behavior for `getOne` against a non-owner (here: anonymous) caller returns 403/404, landing exactly in `_isAuthError`'s trigger set."
    - "app/lib/provider/auth_provider.dart:276-289 (`logout`) — full purge: `jar.deleteAll()`, `_box.removeAll()`, `purgeAccountScopedData(...)`. So the failure mode is not 'silently needs a re-login' but an actual full local session/cache wipe."
  falsification_test: |
    If this hypothesis is correct, correlating server-side logs of
    `authRefresh` failures (any non-2xx/timeout from the PocketBase
    `/api/collections/users/auth-refresh` call inside hooks.server.ts) against
    the timestamps of user-reported logouts should show a hit for most/all
    reports — i.e. every spurious logout is preceded within the same request by
    a failed authRefresh, NOT by a genuinely expired/revoked token. If most
    logout reports occur on a timeline consistent with actual token expiry
    (e.g. always exactly at PocketBase's own token TTL, with no visible upstream
    error), then expiry (candidate 2) rather than error-mishandling would be the
    better explanation instead. Also: server access/error logs around
    `auth-refresh` calls (5xx/timeout counts) during the affected window, and
    whether the PocketBase instance has any history of restarts/high load/rate
    limiting coinciding with report times, would directly confirm the trigger.
  fix_rationale: |
    Not implemented (find_root_cause_only). Fix direction (for the next agent):
    the server should not treat a failed `authRefresh` call as equivalent to an
    invalid token — only an explicit 401 from that call (a truly rejected/expired
    token) should clear the authStore; any other error (network/5xx/timeout)
    should leave the existing authStore and cookie untouched (log and continue
    with the pre-refresh, still-valid state) so a transient hiccup can't nuke a
    real session or force a client-side logout. This directly addresses the root
    cause rather than papering over it in `_isAuthError` (which would just hide
    the symptom locally on one client type while other cookie-bearing sessions
    keep getting wiped).
  blind_spots: |
    Not verified against actual server/device logs — no access to production
    logs, PocketBase restart history, or a tester's device state in this
    session. Have not confirmed the exact PocketBase collection API rule text
    for `users` (assumed owner-only view rule based on standard PocketBase
    defaults and the 403/404 codes `_isAuthError` explicitly guards for) — have
    not read the PocketBase collection rules/migrations to verify this
    assumption directly. Candidate 2 (plain token-expiry-too-short) and
    candidate 4 (OS-level storage eviction) are not fully ruled out as
    additional/alternate contributors, only deprioritized based on code
    evidence gathered.

next_action: |
  Investigation concluded for find_root_cause_only mode. Root cause identified
  with strong file:line evidence spanning web/src/hooks.server.ts (primary
  defect) and the Flutter cookie/auth-restore path (propagation +
  amplification). Returning ROOT CAUSE FOUND to caller; no fix applied per
  scope constraint.

## Evidence

- timestamp: 2026-09-26T00:00:00Z
  checked: app/lib/provider/auth_provider.dart (full file)
  found: |
    `Auth.build()` (42-67) restores session purely from presence of a `pb_auth`
    cookie against the cached `UserEntity.serverUrl`; if the cookie is missing,
    returns null regardless of the still-present ObjectBox `UserEntity` row.
    `_validateInBackground` (85-109) runs unconditionally on every cold start,
    calling `_updateUserEntity` and invoking full `logout()` (276-289) if the
    resulting error is 401/403/404 per `_isAuthError` (133-139). `logout()`
    purges the cookie jar, the ObjectBox `UserEntity`, and account-scoped local
    data — a full wipe, not a soft state.
  implication: |
    The client-side design already anticipates and reacts strongly to
    401/403/404 from the background validation call; the question becomes
    what can produce those codes for an otherwise-valid session.

- timestamp: 2026-09-26T00:05:00Z
  checked: app/lib/provider/cookie_jar_provider.dart, app/lib/main.dart:41-76, app/lib/provider/api_provider.dart (full file)
  found: |
    `PersistCookieJar` uses `FileStorage` in the app documents directory
    (`.cookies` subfolder), `ignoreExpires: false`. The shared Dio client's
    `baseUrl` is `<serverUrl>/api/v1` (the SvelteKit proxy) and has
    `CookieManager(cookieJar)` attached, so every response's Set-Cookie headers
    are auto-persisted to that on-disk jar with no filtering by request type or
    outcome.
  implication: |
    Any Set-Cookie the SvelteKit server sends for `pb_auth` — including one
    that represents a deliberate deletion — is durably written to disk, and the
    entire restore-on-cold-start mechanism trusts that on-disk cookie
    unconditionally.

- timestamp: 2026-09-26T00:12:00Z
  checked: web/src/hooks.server.ts (full `auth` handle, lines 56-187)
  found: |
    Every request loads the authStore from the incoming `pb_auth` cookie (81),
    then — if the loaded token looks valid — calls
    `pb.collection('users').authRefresh({ requestKey: null })` (134-138) inside
    a bare `try { ... } catch (_) { pb.authStore.clear(); ... }` (139-143) with
    no inspection of the failure's cause (real 401 vs network/5xx/timeout).
    `event.locals.user` is assigned from the POST-refresh-attempt authStore
    state (160), so the clear affects the current request too, not just future
    ones. `pb.authStore.exportToCookie(...)` is appended to every response
    unconditionally (180), regardless of whether the store was valid or just
    cleared.
  implication: |
    A single transient failure of the server's own outbound call to PocketBase
    (unrelated to the client's real connectivity or the token's real validity)
    is indistinguishable, in this code, from a genuinely invalid/revoked token.
    It (a) serves the CURRENT request as anonymous and (b) tells the client to
    delete its session cookie.

- timestamp: 2026-09-26T00:18:00Z
  checked: web/node_modules/pocketbase/dist/pocketbase.cjs.js (BaseAuthStore.clear, exportToCookie, getTokenPayload)
  found: |
    `clear()` sets `baseToken=""`, `baseModel=null`. `exportToCookie` computes
    `expires` from `getTokenPayload(this.token)?.exp`; `getTokenPayload("")`
    short-circuits to `{}` (no `exp`), so `expires` falls back to
    `new Date("1970-01-01")` — an already-expired cookie, i.e. a deletion
    instruction per cookie semantics.
  implication: |
    Confirms mechanically that a cleared authStore always serializes to a
    cookie that deletes the client's session, not merely one that omits an
    update.

- timestamp: 2026-09-26T00:24:00Z
  checked: |
    ~/.pub-cache/hosted/pub.dev/cookie_jar-4.0.9/lib/src/jar/default.dart
    (saveFromResponse, lines 140-170) and
    ~/.pub-cache/hosted/pub.dev/cookie_jar-4.0.9/lib/src/jar/persist.dart
    (_save/_filterPathEntries, lines 141-201)
  found: |
    `saveFromResponse` overwrites the existing `pb_auth` map entry with the
    incoming cookie, then immediately removes it if expired (default.dart:
    161-165). `PersistCookieJar._save` calls `_filter`/`_filterPathEntries`
    (persist.dart:124-154), which skips any expired cookie when serializing to
    disk. So the deletion is applied both in memory and durably on disk.
  implication: |
    Confirms the deleting cookie sent by hooks.server.ts actually removes the
    previously-valid `pb_auth` entry from the Flutter app's on-disk cookie
    store, not just from an in-memory cache that would be re-seeded next run.

- timestamp: 2026-09-26T00:30:00Z
  checked: web/src/routes/api/v1/user/[id]/+server.ts (GET), web/src/lib/util/api_util.ts (show())
  found: |
    `_updateUserEntity`'s `GET /user/:id` (the exact call
    `_validateInBackground` makes on every cold start) is served via
    `event.locals.pb.collection('users').getOne(id, ...)` — the same `pb`
    instance whose authStore hooks.server.ts may have just cleared for this
    request.
  implication: |
    If authRefresh failed transiently on this exact request, the GET runs
    anonymously against a collection whose view rule almost certainly requires
    the caller to be the record's own user, matching PocketBase's standard
    404/403 response for a non-owner/anonymous `getOne` — the exact status
    codes `_isAuthError` treats as "log the user out," closing the loop between
    the server-side defect and the client's full local wipe.

- timestamp: 2026-09-26T00:34:00Z
  checked: app/lib/store/account_data_purge.dart (shouldPurgeForIncomingUser)
  found: |
    Pure string-equality check between the cached and incoming user id; only
    fires on an actual id mismatch, and only from within `_updateUserEntity`
    (login/refresh success path), not from `_validateInBackground`'s error
    branch.
  implication: |
    Candidate 5 (account-scoping logic incorrectly clearing a session) is not
    supported by the code — the predicate is correct and narrowly scoped;
    deprioritized as a cause of this symptom.

## Eliminated

- hypothesis: "Account-switching/account-scoping logic (`shouldPurgeForIncomingUser`) incorrectly purges the active session for the same account."
  evidence: "account_data_purge.dart:48-50 is a plain `cachedUserId != null && cachedUserId != incomingUserId` check, only invoked from the success path of `_updateUserEntity`, not from any error/refresh-failure branch. No code path was found where it could misfire for the same account."
  timestamp: 2026-09-26T00:36:00Z
- hypothesis: "Secure/keychain storage misconfiguration wipes the session store on iOS/Android across app termination."
  evidence: "Cookies are persisted via `cookie_jar`'s plain `FileStorage` in the app's documents directory (main.dart:50-59), not Keychain/Keystore-backed secure storage. No OS-level accessibility-class or backup-exclusion code was found that would explain a termination-triggered wipe; the confirmed mechanism (a legitimately-sent deleting Set-Cookie, persisted by ordinary file I/O) fully explains the symptom without invoking storage-backend fragility. Not fully proven absent on-device, so listed as deprioritized rather than fully ruled out."
  timestamp: 2026-09-26T00:38:00Z

## Resolution

root_cause: |
  web/src/hooks.server.ts:134-143 clears the server-side PocketBase authStore
  whenever `pb.collection('users').authRefresh()` throws for ANY reason —
  including transient network/backend failures between the SvelteKit server
  and PocketBase, not just a genuinely invalid/expired token — and then
  unconditionally re-serializes and sends back the (now-cleared, expired)
  `pb_auth` cookie on every response (hooks.server.ts:180). The Flutter app's
  shared Dio client auto-persists that Set-Cookie into its on-disk
  `PersistCookieJar` via `dio_cookie_manager`
  (app/lib/provider/api_provider.dart:61), and the `cookie_jar` package's own
  RFC6265 semantics durably delete the previously-valid `pb_auth` entry from
  disk (cookie_jar-4.0.9 default.dart:161-165, persist.dart:141-154). The same
  request that hit the transient failure is also downgraded to anonymous
  (hooks.server.ts:148-164), so if it happens to be
  `Auth._validateInBackground`'s `GET /api/v1/user/:id` call — which runs on
  EVERY cold start (auth_provider.dart:65,85-109) — PocketBase's own
  view-rule-mismatch behavior returns 403/404, which
  `_isAuthError` (auth_provider.dart:133-139) treats as an invalid session,
  triggering a full local `logout()` (auth_provider.dart:276-289: cookie jar
  wipe + ObjectBox UserEntity removal + account-scoped data purge). On any
  subsequent cold start, `Auth.build()` (auth_provider.dart:42-67) then finds
  no `pb_auth` cookie and reports the user as logged out — even though nothing
  was ever wrong with the user's actual PocketBase credentials or session.
fix:
verification:
files_changed: []
