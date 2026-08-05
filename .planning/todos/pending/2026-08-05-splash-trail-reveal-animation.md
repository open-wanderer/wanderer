---
created: 2026-08-05T00:00:00.000Z
title: Animate the splash trail reveal on the home screen
area: app
files:
  - app/assets/svgs/splash_logo_light.svg
  - app/assets/svgs/splash_logo_dark.svg
  - app/lib/routes/home_screen.dart
  - app/lib/provider/router_provider.dart
  - app/lib/main.dart
  - app/test/routes/splash_logo_asset_test.dart
---

## Problem

`app/lib/routes/home_screen.dart` is the splash shown at `/` while auth resolves. It renders
the two-line logo above a bare `CircularProgressIndicator` — a generic spinner on the one
screen that is every user's first impression of the app, and the anti-reference in `PRODUCT.md`
is precisely "generic SaaS dashboard patterns".

The logo mark already contains the right motif. In `logo_text_twoline_{light,dark}.svg` the
second path is the white trail ribbon:

```
M40.9546 119.979 C53.1847 116.013 64.7988 110.896 67.375 105.25 … 60.75 88.375 … 83.125 77.5 H90.875 … 78.5486 127.669
```

It meets the circle's rim near the bottom-left (~40.9, 120), sweeps right and switchbacks at
~(67, 105), climbs to the mountain baseline at y≈77.5, then returns to the rim at bottom-right
(~78.5, 127.7). Revealing that track from the rim inward to the summit is a loading indicator
that is literally an ascent — on-brand for "The Terrain Log", and it costs no new dependency.

## Decisions taken

**D-1 — Reveal direction: rim → summit, single pass.** Not both ends creeping inward, not a
loop. The trail draws once, from outside to the mountain.

**D-2 — Always complete before exit.** The wait is bounded: `auth_provider.dart:66` caps
validation at `Duration(seconds: 3)`, but a warm start with a cached token typically resolves in
a few hundred ms. A draw sized to the 3s worst case would, in the common case, show ~15% of a
trail and then vanish — an unfinished ascent reads as failure. So: a 1.5s nominal draw, and if
auth resolves early the reveal *accelerates* to the summit before the router moves.

The catch-up bound is tied to the nominal draw, not fixed independently. At 1.5s a warm start
settles auth roughly a fifth of the way up, leaving most of the trail for the catch-up to cover;
capping that at a small fixed number turns the payoff into a blur. It is currently 400ms, scaled
by the fraction still undrawn, so a fast cold start pays ~300ms and a slow one ~120ms. That cost
is accepted deliberately, in exchange for the animation always paying off.

**D-3 — Paint the reveal behind the mark and let the hole do the masking.**

The disc path in the source logo already carries the trail as a *hole*; the white ribbon is a
separate path painted back over it. That inverts the whole problem. Delete the ribbon path and
the hole is open, so **anything drawn behind the picture shows through it already in the shape
of the trail** — the reveal never has to know the trail's geometry at all.

So: a disc-coloured fill behind the mark (which is what makes an unrevealed trail invisible
rather than a window onto the scaffold), and over it a plain circle growing from the trail's
feet on the rim, the whole backdrop clipped to the disc so it cannot spill past the silhouette.
The ribbon's distance from those feet increases monotonically along its length, so the circle
uncovers it end to end, summit last, with no detached islands — verified by rendering the sweep
at 4-unit radius steps.

Rejected: a `ClipPath` walking the two ribbon edges by `PathMetric` (correct, but ~100 lines,
transcribes the edge geometry into Dart, and needs a separate fix for the rim arc bulging below
its chord — which lights a sliver in the disc's bottom edge before anything moves); a
hand-transcribed centreline stroked wide (bleeds across the switchback where the limbs run
close); a `ShaderMask` sweep over the whole mark (reveals mountains and clouds too).

**D-4 — Derive splash-only copies; leave the shipped logos untouched.**
`logo_text_twoline_{light,dark}.svg` is shared by three screens — `home_screen.dart`,
`login_screen.dart:83`, `register_screen.dart:90` — so editing it in place would silently change
login and register to satisfy a splash-only requirement. The derived pair is the source **minus
exactly one path**, nothing recoloured, nothing else touched.

The originals cannot simply be reused as-is: the ribbon is painted *on top* of the disc, and any
cover large enough to hide the unrevealed stretch would also hide the mountains, clouds and
trees sharing that region. The top layer has to be absent from the asset; that cannot be done at
render time.

Because the derivation is a single deletion, the staleness risk is enforced rather than
accepted — `test/routes/splash_logo_asset_test.dart` asserts each derived file equals its source
minus the ribbon path, and pins the path's identity so a reordered redraw cannot delete the
wrong shape. Redesign the logo and the suite fails with instructions to regenerate.

## Work

1. Derive `splash_logo_{light,dark}.svg` from `logo_text_twoline_{light,dark}.svg` by deleting
   the trail ribbon path, without touching the originals. Keep the source `viewBox`
   (`0 0 143 175`) verbatim. No `pubspec.yaml` change needed — `assets/svgs/` is registered as
   a whole directory. The one-line logos are out of scope; the splash does not use them.
2. In `home_screen.dart`, replace the `CircularProgressIndicator` with a `Stack` of
   `CustomPaint` backdrop + the holed logo above it, driven by an `AnimationController`. Pin
   both to an explicit `SizedBox` so the painter's viewBox→pixel scale is a fact rather than an
   assumption about how `SvgPicture` resolves a height-only constraint in a `Stack`. The
   backdrop's two colours must match the SVG's literal fills exactly or the trail seams against
   the disc.
3. Gate the router exit on the reveal. See the accompanying note — the redirect at
   `router_provider.dart:97` is declarative and moves the instant auth flips, so the screen
   needs a flag the redirect can read, and `routerListenable` must notify on that flag without
   regressing its deliberate flip-only filtering.
   Carve out the share entry point: `main.dart`'s `_maybeHandleShare` must release the hold
   immediately. It already skips the auth gate for the same reason (a share is usually a cold
   start, and `_runImportWhenRouterSettled` waits for the router to leave `/`), so holding for
   the animation would reintroduce exactly the splash flash that carve-out exists to avoid.
4. **Failsafe (do not skip).** A gate that can stick strands the user on the *first* screen with
   no way out. Open the gate unconditionally on a wall-clock deadline (~3.5s, just past the auth
   timeout) regardless of animation state, covering a disposed widget, a test env with no vsync,
   and any controller that never reports done.
5. **Reduced motion.** `PRODUCT.md` makes WCAG 2.1 AA + reduced motion a project constraint.
   With `MediaQuery.disableAnimationsOf(context)` true, skip the draw entirely, render the
   completed trail, and open the gate immediately. Follow the existing precedent at
   `app/lib/components/welcome/topography_background.dart:76`.

## Verification

- Cold start with a valid cached token: trail completes, then transition to `/map`.
- Cold start logged out: trail completes, then transition to `/welcome`.
- Airplane mode (auth times out at 3s): trail completes early and holds; redirect fires at 3s.
- OS "reduce motion" on: no draw, no added delay.
- Both themes: at full reveal the mark is indistinguishable from the untouched
  `logo_text_twoline_*`. Watch specifically for a seam along the trail's edge — the backdrop
  colours are hard-coded to match the SVG's fills, so a mismatch shows there first.
- Share a GPX into the app from a cold start: the import screen appears without waiting out the
  reveal.
- **Login and register are visually unchanged.** They share the original asset, which this task
  must not modify — `git status` should show the four logo files clean.

## Status

Implemented 2026-08-05, uncommitted. Geometry verified by rendering the sweep; the reveal's
*timing* (1.5s nominal, ~300ms added to a fast cold start) has not been seen on a device and is
the most likely thing to need tuning.
