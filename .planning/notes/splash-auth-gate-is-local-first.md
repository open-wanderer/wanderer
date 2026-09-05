---
title: The splash auth gate can be answered locally — the network round-trip is the delay
date: 2026-09-05
context: /gsd-explore on the dead time between the splash reveal finishing and the route change
---

## The observation

The trail reveal lands and then the app sits there for a beat before moving to `/map`. The
animation is not the delay; `authProvider` is.

## Why

`GoRouter`'s redirect holds `/` until *both* gates clear
(`app/lib/provider/router_provider.dart:153` and `:164`): the reveal has finished, and
`authProvider` is no longer loading. The reveal is a known 900ms
(`app/lib/routes/home_screen.dart:35`). Auth is not bounded by anything local — `Auth.build()`
awaits a full network validation before it will answer:

```dart
final validation = _updateUserEntity(savedUserEntity.id);
unawaited(validation.catchError((_) => null));
final validated = await validation.timeout(const Duration(seconds: 2));
return validated ?? savedUserEntity;
```

`_updateUserEntity` is `GET /user/:id` with two expands, a settings write and an ObjectBox put.
Up to 2s of it is awaited (`app/lib/provider/auth_provider.dart:66`).

**But the routing decision does not need any of that.** The redirect branches on exactly one
bit: `user != null`. Everything needed to compute that bit was already resolved from disk by
line 63 — a `UserEntity` row exists, and an unexpired `pb_auth` cookie exists. The `await`
throws away an answer it already had.

## Finding 1 — "cookie present" already means "cookie unexpired"

This was the load-bearing discovery, and it means the cheap gate needs no expiry parsing.

`PersistCookieJar` filters expired cookies out on load — `if (cookie.isExpired()) continue;`
in `cookie_jar/lib/src/jar/persist.dart:150`, with the same filter applied in
`jar/default.dart:163`. `isExpired()` checks both `maxAge` and `expires`
(`serializable_cookie.dart:22-28`).

And the server sets a real expiry. `web/src/hooks.server.ts:178` re-issues the cookie on every
response with `pb.authStore.exportToCookie({ httpOnly: false, ... })`, and PocketBase's
`exportToCookie` defaults `expires` to the JWT's own `exp` claim.

So the existing `pbAuthCookie != null` check at `auth_provider.dart:59-61` **is** the unexpired
check. An expired token means the jar returns nothing, which means `return null`, which means
`/welcome` — decided entirely on local disk, with no network call at all. That is the common
staleness case, already handled, already free.

What remains uncovered is server-side revocation inside a still-valid token window (password
change, account deletion). That resolves late, after the route has already moved — see the
crash-surface todo.

## Finding 2 — a fast gate inverts `_catchUp` from rescue to default

`_catchUp` (`app/lib/routes/home_screen.dart:110`) exists for the rare case where auth wins the
race: run the remaining ascent at speed rather than idling on a loaded screen.

Make auth always win and it fires on every launch, a microtask or two into the animation:
`remaining = clamp(300 * (1 - ~0.02), 80, 300)` ≈ **294ms**. The 900ms reveal collapses to a
blink, contradicting the intent recorded at `router_provider.dart:148-152` — *"an unfinished
ascent reads as a failure rather than a load… accepted so the animation always pays off."*

**Decision: retire `_catchUp` entirely.** With auth off the critical path the reveal is the sole
gate, so a fixed 900ms is both the simplest state machine and the predictable brand moment the
animation was built to be. The wall-clock failsafe (`_revealDeadline`, 3500ms) stays — it guards
a reveal that never runs, which is orthogonal.

This inverts the constraint recorded in `splash-exit-is-router-controlled.md`: that note is about
a splash that cannot time its own exit. After this change it is the *only* thing timing it.

## Finding 3 — the background fetch must write back, but filtered

Drop the await and the unawaited validation's result is discarded, while `_box.put()` still
updates the row. Riverpod would keep serving the pre-fetch object: state and DB diverge until
the next `refresh()`.

Writing back unconditionally is the other extreme — 29 call sites `watch(authProvider)`, and two
of them (`provider/trail/trail_search_provider.dart:53`,
`provider/profile/profile_provider.dart:50`) use `watch(authProvider.future)`, so they would
refetch trails and profile a beat after landing on `/map`.

**Decision: write back only on a material difference.** This is the same filter already proven
one layer up in `routerListenable` (`router_provider.dart:88-104`), which was added because
fresh-but-equivalent auth emissions rebuilt the route stack and tore down open modal routes.
Typical launch: zero churn. Genuine web-side change: propagates within a second.

## Rejected

- **Just shorten the 2s timeout to ~600-800ms.** Shrinks the delay instead of removing it, and
  still pays a network round-trip to learn something already on disk.
- **Fast-path unconditionally, accept the `/map` → `/welcome` bounce.** The cookie-expiry filter
  covers the common case for free, so there was no reason to accept the flash for it.
