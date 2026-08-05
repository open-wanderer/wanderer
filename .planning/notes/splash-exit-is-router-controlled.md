---
title: The splash screen does not control its own exit — the router does
date: 2026-08-05
context: /gsd-explore on replacing the home-screen spinner with an animated trail reveal
---

## The constraint

`HomeScreen` (`app/lib/routes/home_screen.dart`, mounted at `/`) has no say in how long it is
shown. The exit is decided entirely by `GoRouter`'s declarative redirect in
`app/lib/provider/router_provider.dart:97-129`:

```dart
if (authState.isLoading && !authState.hasValue) return null;  // hold at /
…
if (!loggedIn)                        return '/welcome';
if (loggedIn && (isAtSplash || …))    return '/map';
```

The redirect re-runs when `routerListenable` notifies, which happens the moment the logged-in
flag flips (`router_provider.dart:73-82`). So the splash is displayed for exactly as long as
`authProvider` is loading — bounded above by the `Duration(seconds: 3)` timeout at
`app/lib/provider/auth_provider.dart:66`, and in the warm-start case a few hundred milliseconds.

**Anything the splash wants to finish before it disappears therefore cannot be timed from
inside the splash.** A widget-local `AnimationController` will simply be torn down mid-flight
when auth resolves. This is the load-bearing fact behind D-2 in the trail-reveal todo
(`.planning/todos/pending/2026-08-05-splash-trail-reveal-animation.md`).

## What a gate has to respect

The fix is a flag the redirect can read — hold at `/` while the splash reports "not done" — but
two existing behaviours constrain how it is wired.

**1. `routerListenable` filters deliberately, and the filter must survive.** It notifies only on
a *flip* of the logged-in boolean:

```dart
if (next.isLoading) return;
final loggedIn = next.value != null;
if (loggedIn == lastLoggedIn) return;
```

The comment above it records why: notifying on every non-loading emission meant a background
re-validation emitting a fresh-but-equivalent `UserEntity` refreshed the router, rebuilt the
route stack, and tore down any open modal route — which broke the share-import flow (see
`app/lib/main.dart:220`). A new gate flag must make the listenable notify when *it* flips,
without widening the auth-state filter back out. Add a second, independent listen; do not
loosen the existing condition.

**2. A stuck gate strands the user on the first screen.** `/` is `initialLocation`. If the flag
never flips — widget disposed mid-animation, a test environment with no vsync, a controller that
never reports completion — there is no navigation affordance on the splash and no way out of
the app short of force-quit. Any gate must be paired with a wall-clock deadline (~3.5s, just
past the auth timeout) that opens it regardless of animation state. Treat the animation as
something that may *shorten* the hold, never as the sole thing that ends it.

## Related

Reduced motion is a project constraint (`PRODUCT.md`: WCAG 2.1 AA + reduced motion), and with
`MediaQuery.disableAnimationsOf(context)` true the gate should open immediately rather than
holding for an animation that will not play. There is an existing precedent for reading that
flag at `app/lib/components/welcome/topography_background.dart:76`.
