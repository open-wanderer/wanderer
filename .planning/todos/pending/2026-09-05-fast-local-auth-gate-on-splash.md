---
created: 2026-09-05T00:00:00.000Z
title: Answer the splash auth gate from local state, not a network round-trip
area: app
files:
  - app/lib/provider/auth_provider.dart
  - app/lib/routes/home_screen.dart
  - app/lib/provider/router_provider.dart
---

## Problem

The trail reveal finishes and the app pauses before moving to `/map`. The redirect holds `/`
until both the reveal has landed *and* `authProvider` has settled
(`app/lib/provider/router_provider.dart:153`, `:164`). The reveal is a fixed 900ms; auth is a
network call awaited for up to 2s (`app/lib/provider/auth_provider.dart:66`), so on anything
but a fast connection auth is the long pole and the user watches a finished animation.

The redirect needs one bit — `user != null` — and `Auth.build()` has already computed it from
disk by line 63 before discarding it behind the `await`.

See `.planning/notes/splash-auth-gate-is-local-first.md` for the full reasoning, including why
the existing cookie check is already an expiry check.

## Changes

**1. `Auth.build()` — return the cached session immediately.**

Keep the local resolution exactly as it is: saved `UserEntity` (line 47) plus an unexpired
`pb_auth` cookie (line 59). `PersistCookieJar` already drops expired cookies on load, and
`hooks.server.ts:178` sets the cookie's `expires` from the JWT `exp` — so cookie-present *is*
token-unexpired, and no new expiry parsing is needed.

Then return `savedUserEntity` without awaiting. Drop the `.timeout(const Duration(seconds: 2))`
and the `TimeoutException` branch with it; the offline case it protected is now the default
path rather than a special one.

**2. `Auth.build()` — keep the validation, unawaited, with two jobs.**

- On success: write back to `state` **only when the fresh entity materially differs** from the
  cached one. Unconditional write-back fires 29 watchers on every cold start and makes the two
  `watch(authProvider.future)` callers (`trail_search_provider.dart:53`,
  `profile_provider.dart:50`) refetch just after landing on `/map`. Use the same
  material-difference filter `routerListenable` already applies at `router_provider.dart:88-104`.
- On auth error: the existing `_isAuthError` → `logout()` path stays. It now runs with `/map`
  live rather than during the splash — see the companion todo on `requireValue` watchers.
- Every other error stays swallowed, preserving the cached session (offline).

**3. `HomeScreen` — retire `_catchUp`, always play the full 900ms.**

With auth off the critical path, `_catchUp` would fire a microtask into every launch and clamp
the reveal to ~294ms, gutting the animation the redirect comment at `router_provider.dart:148`
explicitly says should always pay off. Remove `_catchUp`, `_caughtUp`, `_revealCatchUp`, the
`ref.listen<AsyncValue>(authProvider, ...)` in `build()`, and the post-frame
`if (!ref.read(authProvider).isLoading)` block.

Keep `_revealDeadline` (3500ms) and the reduced-motion early release at
`home_screen.dart:83` — both guard a reveal that never runs, which is a different failure.

**4. Fix the stale comment at `home_screen.dart:45`.** It says "auth_provider.dart's own 3s
validation timeout"; the code was 2s, and after this change there is no awaited timeout at all.
Re-describe `_revealDeadline` as guarding a reveal that never completes.

## Done when

- Cold start with a valid cached session: splash plays a full 900ms and routes to `/map`
  immediately on completion, with no pause, on a slow connection or in airplane mode.
- Cold start with an expired token: routes to `/welcome` with no network call.
- Reduced motion: still exits immediately.
- A username or avatar changed on the web appears without an app restart.
- A typical launch with unchanged user data produces no `authProvider` state emission after the
  initial one (no watcher churn, no trail/profile refetch).
