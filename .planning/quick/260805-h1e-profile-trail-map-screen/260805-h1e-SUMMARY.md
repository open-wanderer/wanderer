---
quick_id: 260805-h1e
slug: profile-trail-map-screen
date: 2026-08-05
status: complete
commits: none (user instruction)
---

# Quick Task 260805-h1e: Profile trail map screen — Summary

## NO COMMITS WERE MADE

The user explicitly instructed mid-run: *"do not commit anything. I want a quick revert
if this goes wrong."* Every change below is **uncommitted and unstaged** in the working
tree. The plan's per-task atomic-commit protocol was deliberately overridden.

**Baseline to revert to:** `0842a918` (working tree was clean apart from the untracked
`.planning/quick/260805-h1e-*/` directory).

**To revert everything:**
```bash
cd /Users/christianbeutel/Documents/svelte/wanderer
git checkout -- app/ web/
rm -rf app/lib/components/map/trail_markers.dart \
       app/lib/provider/trail/profile_trail_bounding_box_provider.dart \
       app/lib/provider/trail/profile_trail_bounding_box_provider.g.dart \
       app/lib/routes/profile_trail_map_screen.dart \
       app/lib/util/map/ \
       app/test/provider/trail/profile_trail_bounding_box_test.dart \
       app/test/routes/profile_trail_map_navigation_test.dart \
       app/test/util/map/ \
       web/src/routes/api/v1/trail/bounding-box/server.test.ts
```
Consequence of the no-commit decision, stated plainly: there are **no per-task rollback
points**. Reverting is all-or-nothing.

## Execution history — two interruptions

Execution did not run cleanly end to end:

1. The executor agent was killed by an **API connection error** partway through Task 4
   (after Tasks 1–3 had landed). It was resumed from its transcript.
2. The resumed agent then **stalled** (no progress for 600s, stream watchdog did not
   recover) partway through Task 5, having written `profile_trail_map_screen.dart` but
   not registered its route or added the entry point.

**The orchestrator completed the remainder of Task 5 directly** — route registration, the
AppBar action button, and the navigation test. All verify gates below were run after that.

## Out-of-scope change reverted by the orchestrator

The executor twice modified `app/lib/theme/icons.dart`, which is in no task's `<files>`
list and had been explicitly excluded. It was reverted to HEAD both times, and the file
is clean at the end of the run.

What it did, and why it mattered — it "fixed" pre-existing FontAwesome deprecation infos
by renaming both the enum members *and* the **map keys** in `fontAwesomeIconsMap`:

```dart
- "facebook-square": FontAwesomeIcons.facebookSquare,
+ "square-facebook":  FontAwesomeIcons.squareFacebook,
```

That map is indexed by strings originating **outside the code** — persisted ObjectBox
values (`app/lib/entities/waypoint_entity.dart:93`), server payloads
(`app/lib/models/converter/fa_icon_data_converter.dart:10`), and `wpt.sym` read from user
GPX files (`app/lib/util/gpx/conversion.dart:540`). Each miss silently falls back to
`FontAwesomeIcons.circle`, so ~15 keys' worth of already-saved waypoints would have
quietly lost their icons with `flutter analyze` and the whole test suite still green.

It also deleted `getTrailIcon`; `git grep` at HEAD confirms that function was already
dead code (defined, never called), so that deletion was harmless — but still unplanned.

The executor's version is preserved at
`<scratchpad>/icons.dart.executor-version` with the diff at `<scratchpad>/icons.dart.diff`.
The deprecation cleanup is defensible on its own merits and could be done later as its own
task — but **without renaming the map keys**.

## What was built

### Task 1 — author-scoped bounding box (D-01, D-05)
`web/src/routes/api/v1/trail/bounding-box/+server.ts` now accepts an optional `handle`
query param. When present it resolves handle→actor via `getActorResponseForHandle` and
ANDs `author = {actor.id}` into all four `multiSearch` queries. Omitting `handle` leaves
the previous behaviour untouched.

Federated (non-local) actors **proxy to the origin instance** per the locked decision:
4s `AbortSignal.timeout`, server-side coordinate validation of the untrusted remote
payload (finite, in-range, min ≤ max), and HTTP 200 + `has_trails: false` on *every*
failure mode — non-200, timeout, unusable payload, actor-resolution failure. No
user-visible error at any layer; the map silently falls back to the world view.

New: `web/src/routes/api/v1/trail/bounding-box/server.test.ts` (6 tests).

### Task 2 — provider family conversion (D-02)
`mapTrailSearchProvider` and `mapClusterSearchProvider` converted to
`(authorId, filterId)` families with named parameters and no defaults, so a transposed
key is a compile error. `keepAlive` retained — `/map` sits in a plain `ShellRoute`, so
`MapScreen` unmounts on tab switch and depends on it.

The author clause is appended inside each provider's `_executeSearch` from the family
key, **not** via `TrailFilter.author` — setting that would have altered the visibility
branch at `app/lib/models/trail.dart:323-329` on a filter object shared with the list
screen. The two endpoints differ in shape: `/search/trails` takes an array (`filter`),
`/search/trails/cluster` takes a string (`filterText`).

All 26 call sites rewritten, including the 11 in `app/test/provider/trail/map_search_deletion_test.dart`.

### Task 3 — bounding-box model and provider
`TrailBoundingBox` in `app/lib/models/trail.dart` corrected — it had camelCase fields
against a snake_case payload, no `has_trails`, and no `fromJson` factory, and had zero
call sites in `app/lib`. New never-throwing `profileTrailBoundingBoxProvider` plus
`app/test/provider/trail/profile_trail_bounding_box_test.dart`.

### Task 4 — shared extractions
`app/lib/components/map/trail_markers.dart` (the unclustered-marker loop) and
`app/lib/util/map/sheet_metrics.dart` (sheet opacity/padding math) extracted from
`map_screen.dart`, with `app/test/util/map/` covering the latter. Only these two pieces
were extracted; the screen was otherwise copied rather than shared, because ~40% of the
layout differs — `/profile/:handle` is outside the bottom-nav shell, making every
`kBottomNavigationBarHeight` offset wrong there.

### Task 5 — the screen (completed by the orchestrator)
`app/lib/routes/profile_trail_map_screen.dart` (988 lines): a gate-then-build split —
outer `ConsumerWidget` resolves `profileProvider(handle)` and only then builds the inner
`ConsumerStatefulWidget` with a non-null `authorId`, guaranteeing the family key is
stable for the view's whole lifetime so an unscoped search can never fire mid-resolution
and `dispose()` has an unambiguous key to invalidate.

Note this outer/inner split is a **new pattern** — `list_detail_map_screen.dart` is a
single `ConsumerStatefulWidget` with no such split. The plan originally miscited it as a
precedent; that was corrected before execution.

Wired up by the orchestrator:
- Nested `GoRoute(path: 'map')` under the existing `'trails'` child of `/profile/:handle`
  in `app/lib/provider/router_provider.dart`, mirroring the `/list/:id/map` precedent.
- AppBar `actions` entry in `app/lib/routes/profile_trail_screen.dart` —
  `FontAwesomeIcons.mapLocationDot`, tooltip `l10n.map` (existing ARB key, no 12-language
  churn), pushing `/profile/${widget.handle}/trails/map`.
- `app/test/routes/profile_trail_map_navigation_test.dart` (3 tests), modelled on
  `profile_trail_screen_navigation_test.dart`: a real `GoRouter` widget test asserting the
  router's actual location, not a source-grep. Note `find.widgetWithIcon` does **not**
  work here — `FaIconData` is not an `IconData` in this codebase — so the finder is
  `find.byTooltip('Map')`.

## Verify gate results (all run, all real)

| Gate | Result |
|---|---|
| `flutter analyze` | **0 errors, 0 warnings** (36 pre-existing infos, all in `lib/theme/icons.dart` + one dangling doc comment in `lib/util/local/id.dart` — none introduced here) |
| `flutter test` (full suite) | **1020 passed, 1 skipped** |
| `flutter test test/routes/profile_trail_map_navigation_test.dart` | **3 passed** |
| `npx vitest run .../bounding-box/server.test.ts` | **6 passed** |
| Task 4 symbol gate (3× `grep -q`) | **PASS** — all three extracted symbols wired into `map_screen.dart` |

Note the Task 4 gate was amended before execution: the original `grep -c … == 0` form
could only fail when the count was zero, so one wired symbol out of three would have
passed. It is now three separate `grep -q` calls ANDed together.

## KNOWN COVERAGE GAP — read this before shipping

**There is no test anywhere covering `app/lib/routes/map_screen.dart`.** The family
conversion rewrote ~15 provider call sites in that file and **none of them have any CI
signal**. `flutter analyze` will catch a type error, but it cannot catch a *transposed or
wrong family key* — e.g. passing the profile's `authorId` where the global map's `null`
belongs. That would compile, pass every test, and silently show the wrong trails on the
main map.

This is the single largest risk in the change.

## <human-check> — verify on device

1. **`/map` regression (highest priority).** Open the main map tab. Confirm it still shows
   trails from all authors, that pan/zoom re-searches, that clusters expand on tap, that
   the bottom sheet lists the same trails as the pins, and that "search this area" works.
   This is the path with no automated coverage.
2. **Profile map, own profile.** Open a profile → Trails → the new map action. Confirm it
   opens fitted to your trails' bounding box and shows only your trails. Private trails
   should appear here (correct — governed by the shared filter's public/private toggles).
3. **Profile map, another local user's profile.** Confirm only their trails appear and no
   private trail of theirs is visible.
4. **Camera isolation.** Move the main map, open a profile map, come back. The main map
   must be exactly where you left it, and the profile map must open at the bbox fit every
   time (it deliberately keeps no camera state).
5. **Shared filter.** Set a filter in the profile trail *list*, open the map — the pins
   should already reflect it. Change it on the map, pop back — the list should reflect the
   change. This is intended behaviour per the locked decision.
6. **Federated profile.** Open a remote profile's trail map. It will most likely fall back
   to the world view (the proxy only succeeds against remote instances running this same
   new param), and it must do so **silently** — no error, no spinner hang. Pins should
   still appear after panning or "search this area", since remote trails are in the local
   index.
7. **Degenerate cases.** A profile with exactly one trail (zero-area bbox) and a profile
   with no trails (`has_trails: false`) — neither should crash or zoom absurdly.

## Deferred / not done

- The 100-result cap (`hitsPerPage`) is **unchanged**, per the locked scope. Zoomed out
  far enough, a profile with more than 100 trails in view shows an arbitrary 100 with no
  "showing N of M" indication. This is the known limitation the exploration session
  identified and deliberately deferred.
- Meilisearch `maxTotalHits` untouched.
- No line/heatmap tile pipeline.
- Own-profile unsynced local ObjectBox trails do **not** appear on the map (they are not
  in Meilisearch). The list screen merges them; the map does not. Explicitly permitted by
  CONTEXT.md — no merge was built.
