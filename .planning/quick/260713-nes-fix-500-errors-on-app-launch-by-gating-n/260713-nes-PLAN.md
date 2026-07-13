---
phase: quick-260713-nes
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/entities/local_settings_entity.dart
  - app/lib/objectbox.g.dart
  - app/lib/objectbox-model.json
  - app/lib/provider/auth_provider.dart
autonomous: false
requirements: [QUICK-260713-nes]

must_haves:
  truths:
    - "On a valid cached session with network available, launch validates the session BEFORE navigating to /map, so the ~20 screen-level providers never race an unconfirmed session (no concurrent 500 burst)."
    - "On launch with no/slow network, the app falls back to the cached user within ~3s and offline features remain fully usable (offline login unchanged)."
    - "A validation call returning 401/403/404 logs the user out and routes to /welcome (existing behavior preserved)."
    - "Repeated 500-class failures from the validation call across separate launches log the user out only after 3 consecutive failures; a single transient 500 preserves the session."
    - "Any successful validation resets the consecutive-failure counter to 0."
  artifacts:
    - path: "app/lib/entities/local_settings_entity.dart"
      provides: "Persistent device-local consecutiveAuthValidationFailures counter"
      contains: "consecutiveAuthValidationFailures"
    - path: "app/lib/provider/auth_provider.dart"
      provides: "Timeout-gated build() validation + server-error self-heal"
      contains: "timeout"
  key_links:
    - from: "app/lib/provider/auth_provider.dart::build()"
      to: "_updateUserEntity"
      via: ".timeout(const Duration(seconds: 3))"
      pattern: "timeout\\(const Duration\\(seconds: 3\\)\\)"
    - from: "app/lib/provider/auth_provider.dart::build()"
      to: "logout()"
      via: "deferred microtask on terminal auth error / failure threshold"
      pattern: "Future\\.microtask|unawaited"
    - from: "app/lib/provider/auth_provider.dart::_updateUserEntity"
      to: "LocalSettingsEntity counter"
      via: "reset to 0 after successful put"
      pattern: "consecutiveAuthValidationFailures"
---

<objective>
Eliminate the launch-time burst of 500 errors that appears when the app resumes a cached session, while keeping offline login working exactly as today, and make a genuinely broken session self-heal instead of failing silently on every launch.

Root cause (already diagnosed, do NOT re-investigate): `Auth.build()` returns the cached session instantly and fires `_updateUserEntity()` as a fully `unawaited()` background validation call. The router treats "auth not loading" as "logged in" and immediately navigates to `/map`, which mounts ~20+ screen-level providers that all fire concurrent authenticated API calls on the shared Dio instance while the background validation is also touching the session/user record — a SQLite-contention thundering herd that surfaces as 500s. `_isAuthError()` only logs out on 401/403/404, so a genuinely broken session (persistent 500 from the validation call) never self-corrects.

Fix, contained entirely to `Auth.build()`'s internal timing/error handling plus a small persisted counter:
1. Gate navigation on the validation call with a short timeout (offline-safe fallback).
2. Broaden self-healing so repeated 500s from the validation call trigger logout, without false-positiving on a single transient hiccup.

Purpose: Remove the concurrent-burst trigger in the common (online) case and give the app a recovery path, without changing offline semantics.
Output: A timeout-gated, self-healing `Auth.build()` backed by a persistent consecutive-failure counter.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# Files to modify / consult (already read during planning — re-read only if needed)
@app/lib/provider/auth_provider.dart
@app/lib/entities/local_settings_entity.dart
@app/lib/provider/local_settings_provider.dart

# Context only (DO NOT modify — the fix is contained to auth_provider.dart + the counter field)
# app/lib/provider/router_provider.dart  — how navigation reacts to auth state (redirect gate at lines 76-108)
# app/lib/provider/api_provider.dart      — shared Dio instance
# app/lib/entities/user_entity.dart       — cached session entity shape

Key existing facts confirmed during planning:
- `_updateUserEntity(id)` is the single validation call; it throws a `DioException` on non-2xx
  (statusCode 401/403/404 for a rejected session, 500 for the contention case) and writes the
  `UserEntity` to ObjectBox on success. It is reused by `login()`, `refresh()` and `build()`.
- `_isAuthError(err)` (statusCode == 401 || 403 || 404) MUST stay unchanged; add a sibling check
  for server errors — do NOT fold 500 into `_isAuthError`.
- `TimeoutException` is already available via the existing `import 'dart:async'`.
- `LocalSettingsEntity` is a device-local, single-row ObjectBox entity (get-or-create pattern:
  `box.getAll().firstOrNull ?? LocalSettingsEntity()`). It survives app kills and is NOT clobbered
  by server data — the correct home for a persistent counter. It currently holds only `themeMode`.
- `router_provider.dart` needs NO changes: its redirect already stays on `/` (splash) while auth
  `isLoading && !hasValue`, and navigates once auth resolves. Gating build() simply keeps the splash
  visible until validation resolves or the timeout fires.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add a persistent device-local validation-failure counter</name>
  <files>app/lib/entities/local_settings_entity.dart, app/lib/objectbox.g.dart, app/lib/objectbox-model.json</files>
  <action>
Add an `int consecutiveAuthValidationFailures` field to `LocalSettingsEntity`, defaulting to `0`, wired through the constructor as a named optional parameter (`this.consecutiveAuthValidationFailures = 0`) exactly like the existing `themeMode` field. This is a device-local counter that must survive full app kills, so it belongs on this persisted entity (not on `UserEntity`, which is rewritten from server JSON on every successful validation and removed on logout).

Then regenerate the ObjectBox bindings by running build_runner from the `app/` directory: `dart run build_runner build --delete-conflicting-outputs`. This updates the generated `objectbox.g.dart` bindings and `objectbox-model.json` model (new property id/uid) so the field is persisted; ObjectBox auto-migrates the added column with its default. Do NOT hand-edit either generated file — let build_runner produce them.

Do not change any other field, the entity id strategy, or the existing get-or-create usage in `local_settings_provider.dart`.
  </action>
  <verify>
    <automated>cd app && dart run build_runner build --delete-conflicting-outputs && grep -q consecutiveAuthValidationFailures lib/objectbox.g.dart && flutter analyze lib/entities/local_settings_entity.dart</automated>
  </verify>
  <done>`LocalSettingsEntity` exposes `int consecutiveAuthValidationFailures` (default 0); the regenerated `objectbox.g.dart` and `objectbox-model.json` include the new property; `flutter analyze` reports no errors for the entity file.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Gate build() on timeout-bounded validation + self-heal on repeated 500s</name>
  <files>app/lib/provider/auth_provider.dart</files>
  <behavior>
    - Online, fast network: build() awaits validation, returns the VALIDATED user (already written to ObjectBox by `_updateUserEntity`) before /map mounts — no cached-but-unconfirmed return.
    - Offline / slow network (`.timeout` fires, OR Dio connection/connectivity error): build() returns the cached `savedUserEntity` unchanged; the failure counter is NOT incremented. Offline login must keep working exactly as before.
    - Validation returns 401/403/404 (`_isAuthError` true): user is logged out and build() returns null (router redirects to /welcome).
    - Validation returns a 500-class error (statusCode >= 500): increment the persistent counter; if it reaches the threshold (3) log out and return null; otherwise (1st/2nd consecutive 500) preserve the cached session by returning `savedUserEntity`.
    - Any successful validation resets the counter to 0.
  </behavior>
  <action>
Rewrite ONLY the `pbAuthCookie != null` branch of `build()` (currently the `unawaited(_updateUserEntity(...).catchError(...))` block) and add the small supporting helpers. Preserve the public method signatures of `register`, `login`, `refresh`, and `logout` — change only `build()` internals and add `_isServerError` + counter helpers. `build()`'s own signature (`FutureOr<UserEntity?> build()`) stays unchanged, so no re-run of build_runner is required for auth_provider (verify with analyze; only re-run if analyze reports a generated-code mismatch).

Replace the optimistic unawaited fire-and-forget with a timeout-gated await:
- Kick off `final validation = _updateUserEntity(savedUserEntity.id);`.
- Attach a no-op error sink to the SOURCE future so a late failure that arrives AFTER the timeout has already fired does not surface as an unhandled async error: `unawaited(validation.catchError((_) => null));` (rationale: `.timeout` does NOT cancel the underlying Dio request; without this sink a post-timeout rejection becomes an unhandled future error). Then `await validation.timeout(const Duration(seconds: 3))` inside a try/catch and branch on the outcome:
  - Success -> return the validated user (fall back to `savedUserEntity` if the call somehow returns null): `return validated ?? savedUserEntity;`.
  - `on TimeoutException` -> return `savedUserEntity` (offline/slow fallback; do NOT touch the counter). This is a hard requirement: offline use is a core scenario for this hiking app.
  - `catch (err)`: if `_isAuthError(err)` -> trigger a deferred logout and return null. Else if `_isServerError(err)` -> bump the persistent counter; if the new value >= the threshold constant (`3`) trigger a deferred logout and return null, otherwise return `savedUserEntity`. Else (any other error, e.g. Dio `connectionError`/`connectionTimeout` — offline) -> return `savedUserEntity`.

Add `bool _isServerError(Object err)`: returns true when `err is DioException` and `err.response?.statusCode != null && statusCode >= 500` (covers the reported 500 and other 5xx). Keep `_isAuthError` exactly as-is (401/403/404).

Deferred logout (CRITICAL — do NOT call `logout()` synchronously inside `build()`): `logout()` assigns `state = const AsyncLoading()` and calls `ref.invalidateSelf()`, which is illegal while `build()` is still running. Schedule it instead — e.g. `Future.microtask(logout);` (or `unawaited(Future(() => logout()));`) — and return `null` from `build()` on the same pass. Returning null means the router treats the session as logged-out immediately (no /map navigation, no burst), and the scheduled `logout()` performs the cookie/box cleanup + invalidateSelf afterward. This preserves the existing logout OUTCOME (cookies deleted, box cleared, redirect to /welcome) while staying build()-safe.

Add a small private threshold constant (e.g. `static const _maxConsecutiveValidationFailures = 3;`) and three counter helpers backed by the `LocalSettingsEntity` box (mirror the get-or-create read-modify-write pattern in `local_settings_provider.dart`, and ONLY mutate `consecutiveAuthValidationFailures` so `themeMode` is preserved):
  - a read/bump helper that loads the single row (or a fresh `LocalSettingsEntity()` if none), increments `consecutiveAuthValidationFailures`, `put`s it, and returns the new value.
  - a reset helper that loads the single row, sets the counter to 0, and `put`s it (no-op-safe if the row is absent/already 0).
Call the reset helper inside `_updateUserEntity` immediately after the successful `_box.put(userEntity)` (before `return userEntity;`) so EVERY successful validation — from `build()`, `login()`, or `refresh()` — resets the counter, matching "reset on any successful validation." Bumping happens only in the `build()` 500 branch.

Do NOT build a generic retry/backoff framework, do NOT change `api_provider.dart`, `router_provider.dart`, or any screen-level provider — the broadened 500->logout logic applies ONLY to this one dedicated validation call.
  </action>
  <verify>
    <automated>cd app && grep -q 'timeout(const Duration(seconds: 3))' lib/provider/auth_provider.dart && grep -q '_isServerError' lib/provider/auth_provider.dart && grep -q 'consecutiveAuthValidationFailures' lib/provider/auth_provider.dart && flutter analyze lib/provider/auth_provider.dart</automated>
  </verify>
  <done>`build()` awaits `_updateUserEntity(...).timeout(const Duration(seconds: 3))` with distinct handling for success / `TimeoutException` (offline fallback) / auth error (deferred logout) / 5xx (bump + threshold logout); `_updateUserEntity` resets the counter on success; `_isAuthError` unchanged and `_isServerError` added; `register`/`login`/`refresh`/`logout` signatures untouched; `flutter analyze lib/provider/auth_provider.dart` reports no errors.</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <what-built>
Timeout-gated, self-healing session validation in `Auth.build()`: launch now validates the cached session (up to 3s) before navigating to /map, falls back to the cached user offline, and logs out on 401/403/404 or after 3 consecutive 500s from the validation call. Automated `flutter analyze` and build_runner regeneration already passed; this checkpoint confirms the four runtime behaviors on a device/emulator, matching how this app verifies device behavior.
  </what-built>
  <how-to-verify>
Run the app (`cd app && flutter run`) against a real Wanderer server with a valid cached session, then confirm each scenario:

1. Online happy path: cold-launch with network available and a valid cached session. Expect launch to feel effectively instant (brief splash on `/`, then `/map`) with NO burst of 500 errors in the console/network trace as `/map`'s providers mount.
2. Offline fallback (the non-negotiable one): enable airplane mode, then cold-launch with the same valid cached session. Expect the splash to resolve within ~3s to the cached user and offline features to remain usable — no hang, no forced logout.
3. Real auth rejection: force the validation call to return 401/403/404 (e.g. invalidate the server-side session for this user, or point at a server that rejects the cookie), then cold-launch. Expect automatic logout and redirect to /welcome.
4. Persistent server error self-heal: make the `/user/:id` validation endpoint return 500 (e.g. stop the DB/backend so PocketBase 500s, or otherwise force a 5xx on that call). Cold-launch THREE times. Expect the session to be preserved on launches 1 and 2, and on the 3rd consecutive 500 the app logs out and redirects to /welcome (self-heal) rather than staying stuck. Then restore the server, log in, and confirm a subsequent successful launch does not immediately log out (counter reset).
  </how-to-verify>
  <resume-signal>Type "approved" once all four scenarios behave as described, or describe any deviation (especially any lingering 500 burst in scenario 1 or any offline regression in scenario 2).</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Flutter client → PocketBase server | Session cookie (`pb_auth`) crosses here on the `/user/:id` validation call; server responses (2xx/4xx/5xx) drive client auth state. Pre-existing boundary; this change alters only client-side reaction timing/logic. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-nes-01 | Denial of Service | `Auth.build()` self-heal on 5xx | accept | A server (or MITM) returning 500 to the validation call can force a logout, but only after 3 consecutive failures across launches and only via this one call; the counter resets on any success. Bounded, intentional self-heal — not a new attack surface. |
| T-nes-02 | Spoofing / stale session | `_updateUserEntity` timeout fallback | mitigate | Offline fallback returns the cached user without server confirmation (required for offline use), but online launches now confirm the session before mounting authenticated screens, shrinking the window a stale/spoofed cookie is trusted. 401/403/404 still forces logout. |
| T-nes-SC | Tampering | package installs | accept | No new packages added; only an ObjectBox field + build_runner regeneration of existing generated code. No supply-chain surface. |
</threat_model>

<verification>
- `cd app && dart run build_runner build --delete-conflicting-outputs` completes without errors (Task 1).
- `cd app && flutter analyze lib/entities/local_settings_entity.dart lib/provider/auth_provider.dart` reports no errors.
- Grep gates pass: `.timeout(const Duration(seconds: 3))`, `_isServerError`, and `consecutiveAuthValidationFailures` all present in `auth_provider.dart`; `consecutiveAuthValidationFailures` present in regenerated `objectbox.g.dart`.
- Public signatures of `register`, `login`, `refresh`, `logout` unchanged (diff confirms only `build()` internals + new private helpers/const changed).
- Human-verify checkpoint confirms the four runtime behaviors (online no-burst, offline fallback within ~3s, 401 logout, 3rd-consecutive-500 logout + counter reset).
</verification>

<success_criteria>
- Online launch with a valid cached session validates before navigating to /map, with no concurrent 500 burst.
- Offline/slow launch falls back to the cached user within ~3s — offline login and features unchanged.
- 401/403/404 from the validation call still logs out and redirects to /welcome.
- Three consecutive 500s from the validation call (across launches) trigger logout; a single 500 does not; success resets the counter.
- Change is contained to `auth_provider.dart` internals + the `LocalSettingsEntity` counter field; no changes to router/api/screen providers; generated ObjectBox bindings regenerated via build_runner.
</success_criteria>

<output>
Create `.planning/quick/260713-nes-fix-500-errors-on-app-launch-by-gating-n/260713-nes-SUMMARY.md` when done.
</output>
